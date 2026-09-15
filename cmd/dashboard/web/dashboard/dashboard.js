const search = document.getElementById("search");
const containers = document.getElementById("containers");
const empty = document.getElementById("empty");
const refreshButton = document.getElementById("refresh");
const refreshTime = document.getElementById("refresh-time");
const sortButtons = document.querySelectorAll(".sort-button");
const sortHeaders = document.querySelectorAll(
    "th[data-sort-column]"
);

let currentMetadata = {
    packages: []
};

let sortState = {
    column: "image",
    direction: "ascending"
};

function render(metadata) {
    currentMetadata = metadata;

    document.getElementById("repository").textContent =
        metadata.repository || "unknown";

    document.getElementById("repository").href =
        metadata.url || "#";

    document.getElementById("container-count").textContent =
        metadata.packages.length;

    document.getElementById("last-updated").textContent =
        metadata.last_updated || "unknown";

    document.getElementById("updated-count").textContent =
        metadata.packages.filter(
            container => container.latest_updated
        ).length;

    renderContainers();
    updateRefreshTime();
}

function renderContainers() {
    const packages = [...currentMetadata.packages];

    packages.sort(compareContainers);

    containers.innerHTML = "";

    for (const container of packages) {
        const row = document.createElement("tr");

        row.innerHTML = `
            <td>
                <div class="container-name">
                    ${escapeHtml(container.image)}
                </div>
            </td>

            <td>
                <span class="version">
                    ${escapeHtml(
            container.latest || "unknown"
        )}
                </span>
            </td>

            <td>
                <span class="updated">
                    ${escapeHtml(
            container.latest_updated || "—"
        )}
                </span>
            </td>

            <td>
                ${container.latest_digest
                ? `<code>${escapeHtml(
                    container.latest_digest
                )}</code>`
                : `<span class="unknown">—</span>`
            }
            </td>

            <td class="actions">
                <a
                    href="${escapeAttribute(
                container.url
            )}"
                    target="_blank"
                    rel="noopener noreferrer"
                    title="Open on GitHub"
                >
                    ↗
                </a>
            </td>
        `;

        containers.appendChild(row);
    }

    updateSearch();
    updateSortIndicators();
}

function compareContainers(left, right) {
    const leftValue = getSortValue(left);
    const rightValue = getSortValue(right);

    let result = 0;

    if (sortState.column === "latest_updated") {
        result = compareDates(
            leftValue,
            rightValue
        );
    } else {
        result = leftValue.localeCompare(
            rightValue,
            undefined,
            {
                numeric: true,
                sensitivity: "base"
            }
        );
    }

    if (sortState.direction === "descending") {
        return result * -1;
    }

    return result;
}

function getSortValue(container) {
    switch (sortState.column) {
        case "latest":
            return container.latest || "";

        case "latest_updated":
            return container.latest_updated || "";

        case "image":
        default:
            return container.image || "";
    }
}

function compareDates(left, right) {
    const leftTime = left
        ? Date.parse(left)
        : Number.NEGATIVE_INFINITY;

    const rightTime = right
        ? Date.parse(right)
        : Number.NEGATIVE_INFINITY;

    return leftTime - rightTime;
}

function setSort(column) {
    if (sortState.column === column) {
        sortState.direction =
            sortState.direction === "ascending"
                ? "descending"
                : "ascending";
    } else {
        sortState.column = column;

        sortState.direction =
            column === "latest_updated"
                ? "descending"
                : "ascending";
    }

    renderContainers();
}

function updateSortIndicators() {
    for (const header of sortHeaders) {
        const column = header.dataset.sortColumn;

        const indicator = header.querySelector(
            ".sort-indicator"
        );

        if (column === sortState.column) {
            header.setAttribute(
                "aria-sort",
                sortState.direction
            );

            indicator.textContent =
                sortState.direction === "ascending"
                    ? "▲"
                    : "▼";
        } else {
            header.removeAttribute("aria-sort");

            indicator.textContent = "◆";
        }
    }
}

function updateSearch() {
    const query = search.value
        .trim()
        .toLowerCase();

    const rows = containers.querySelectorAll("tr");

    let visible = 0;

    for (const row of rows) {
        const name = row
            .querySelector(".container-name")
            .textContent
            .toLowerCase();

        const matches = name.includes(query);

        row.hidden = !matches;

        if (matches) {
            visible++;
        }
    }

    empty.hidden = visible !== 0;
}

function updateRefreshTime() {
    refreshTime.textContent =
        "Dashboard checked " +
        new Date().toLocaleTimeString();
}

async function refresh() {
    refreshButton.disabled = true;
    refreshButton.textContent = "↻ Refreshing...";

    try {
        const response = await fetch(
            "/api/metadata",
            {
                cache: "no-store"
            }
        );

        if (!response.ok) {
            throw new Error(
                `HTTP ${response.status}`
            );
        }

        const responseData = await response.json();

        render(responseData.metadata);
    } catch (error) {
        console.error(
            "Failed to refresh metadata:",
            error
        );
    } finally {
        refreshButton.disabled = false;
        refreshButton.textContent = "↻ Refresh";
    }
}

function escapeHtml(value) {
    const div = document.createElement("div");

    div.textContent = String(value ?? "");

    return div.innerHTML;
}

function escapeAttribute(value) {
    return escapeHtml(value)
        .replace(/"/g, "&quot;");
}

function loadInitialMetadata() {
    const rows = Array.from(
        document.querySelectorAll("#containers tr")
    );

    return {
        packages: rows.map(row => ({
            image: row
                .querySelector(".container-name")
                .textContent
                .trim(),

            latest: row
                .querySelector(".version")
                .textContent
                .trim(),

            latest_updated: row
                .querySelector(".updated")
                .textContent
                .trim(),

            latest_digest: row
                .querySelector("code")
                ?.textContent
                .trim() || "",

            url: row
                .querySelector(".actions a")
                ?.href || ""
        }))
    };
}

for (const button of sortButtons) {
    button.addEventListener(
        "click",
        () => setSort(button.dataset.sort)
    );
}

search.addEventListener(
    "input",
    updateSearch
);

refreshButton.addEventListener(
    "click",
    refresh
);

setInterval(
    refresh,
    60 * 1000
);

currentMetadata = loadInitialMetadata();

renderContainers();
updateRefreshTime();