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

    <td>
        <div class="tag-list">
            ${renderTags(container.tags)}
        </div>
    </td>

    <td class="actions">
        <details class="tag-explorer">
            <summary
                title="Explore tags"
                aria-label="Explore tags for ${escapeAttribute(
                container.image
            )}"
            >
                ☰
            </summary>

            <div class="tag-explorer-panel">
                <div class="tag-explorer-header">
                    <strong>
                        ${escapeHtml(container.image)}
                    </strong>

                    <a
                        href="${escapeAttribute(
                container.url
            )}"
                        target="_blank"
                        rel="noopener noreferrer"
                    >
                        Open on GitHub ↗
                    </a>
                </div>

                <h3>Current tags</h3>

                <div class="tag-list">
                    ${renderTags(container.tags)}
                </div>

                <h3>Tag history</h3>

                ${renderPastVersions(
                container.past_versions
            )}
            </div>
        </details>
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

function closeTagExplorers(except = null) {
    const explorers = document.querySelectorAll(
        ".tag-explorer[open]"
    );

    for (const explorer of explorers) {
        if (explorer === except) {
            continue;
        }

        explorer.open = false;
    }
}

function handleDocumentClick(event) {
    const explorer = event.target.closest(
        ".tag-explorer"
    );

    if (explorer) {
        return;
    }

    closeTagExplorers();
}

function renderTags(tags) {
    if (!tags || tags.length === 0) {
        return `<span class="unknown">—</span>`;
    }

    return tags.map(tag => `
        <span class="tag ${tag === "latest" ? "tag-latest" : ""}">
            ${escapeHtml(tag)}
        </span>
    `).join("");
}

function renderPastVersions(pastVersions) {
    if (!pastVersions || pastVersions.length === 0) {
        return `
            <p class="unknown">
                No historic tags found.
            </p>
        `;
    }

    return `
        <table class="history-table">
            <thead>
                <tr>
                    <th>Tag</th>
                    <th>Published</th>
                    <th>Digest</th>
                </tr>
            </thead>

            <tbody>
                ${pastVersions.map(version => `
                    <tr>
                        <td>
                            <code>${escapeHtml(version.tag)}</code>
                        </td>

                        <td>
                            ${escapeHtml(version.release_date)}
                        </td>

                        <td>
                            <code>
                                ${escapeHtml(version.digest || "—")}
                            </code>
                        </td>
                    </tr>
                `).join("")}
            </tbody>
        </table>
    `;
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

document.addEventListener(
    "click",
    handleDocumentClick
);

document.addEventListener(
    "toggle",
    event => {
        const explorer = event.target;

        if (
            !explorer.matches(".tag-explorer") ||
            !explorer.open
        ) {
            return;
        }

        closeTagExplorers(explorer);
    },
    true
);

document.addEventListener(
    "keydown",
    event => {
        if (event.key !== "Escape") {
            return;
        }

        closeTagExplorers();
    }
);

setInterval(
    refresh,
    60 * 1000
);

refresh();
