import { Events } from "@wailsio/runtime";
import * as Journal from "./bindings/changeme/backend/bindings/journalservice.js";
import * as Models from "./bindings/changeme/backend/models/models.js";

const timeElement = document.getElementById("time");
let equityChart = null;
let allTrades = [];
let pagination = {
  currentPage: 1,
  pageSize: 25,
  totalTrades: 0,
  totalPages: 1,
};

// Connection test functionality
const pingBtn = document.getElementById("ping-btn");
const pingResult = document.getElementById("ping-result");
const statusIndicator = document.getElementById("status-indicator");

if (pingBtn && pingResult && statusIndicator) {
  pingBtn.addEventListener("click", async () => {
    try {
      const v = await Journal.Ping();
      pingResult.innerText = `Connected - Version: ${v}`;
      statusIndicator.classList.add("connected");
    } catch (e) {
      pingResult.innerText = "Connection failed";
      statusIndicator.classList.remove("connected");
    }
  });
}

// Global query state
let currentQuery = Models.Query.createFrom({
  symbol: "",
  side: "",
  startTime: undefined,
  endTime: undefined,
  limit: 10000, // Get all trades for client-side pagination
  offset: 0,
});

// Helper function to build query from filters
function buildQueryFromFilters() {
  const symbol = document.getElementById("filter-symbol")?.value || "";
  const side = document.getElementById("filter-side")?.value || "";
  const start = document.getElementById("filter-start")?.value || "";
  const end = document.getElementById("filter-end")?.value || "";

  currentQuery = Models.Query.createFrom({
    symbol: symbol.trim(),
    side: side,
    startTime: start ? new Date(start).toISOString() : undefined,
    endTime: end ? new Date(end).toISOString() : undefined,
    limit: 10000,
    offset: 0,
  });

  return currentQuery;
}

// Pagination functions
function updatePaginationInfo() {
  const info = document.getElementById("pagination-info");
  const start = (pagination.currentPage - 1) * pagination.pageSize + 1;
  const end = Math.min(
    pagination.currentPage * pagination.pageSize,
    pagination.totalTrades
  );

  if (pagination.totalTrades === 0) {
    info.textContent = "No trades found";
  } else {
    info.textContent = `Showing ${start}-${end} of ${pagination.totalTrades} trades`;
  }
}

function updatePaginationButtons() {
  const container = document.getElementById("pagination-buttons");
  if (!container) return;

  container.innerHTML = "";

  if (pagination.totalPages <= 1) return;

  // Previous button
  const prevBtn = document.createElement("button");
  prevBtn.textContent = "Previous";
  prevBtn.disabled = pagination.currentPage === 1;
  prevBtn.addEventListener("click", () => {
    if (pagination.currentPage > 1) {
      pagination.currentPage--;
      renderCurrentPage();
    }
  });
  container.appendChild(prevBtn);

  // Page numbers
  const maxVisiblePages = 5;
  let startPage = Math.max(
    1,
    pagination.currentPage - Math.floor(maxVisiblePages / 2)
  );
  let endPage = Math.min(
    pagination.totalPages,
    startPage + maxVisiblePages - 1
  );

  if (endPage - startPage < maxVisiblePages - 1) {
    startPage = Math.max(1, endPage - maxVisiblePages + 1);
  }

  for (let i = startPage; i <= endPage; i++) {
    const pageBtn = document.createElement("button");
    pageBtn.textContent = i;
    pageBtn.className = i === pagination.currentPage ? "active" : "";
    pageBtn.addEventListener("click", () => {
      pagination.currentPage = i;
      renderCurrentPage();
    });
    container.appendChild(pageBtn);
  }

  // Next button
  const nextBtn = document.createElement("button");
  nextBtn.textContent = "Next";
  nextBtn.disabled = pagination.currentPage === pagination.totalPages;
  nextBtn.addEventListener("click", () => {
    if (pagination.currentPage < pagination.totalPages) {
      pagination.currentPage++;
      renderCurrentPage();
    }
  });
  container.appendChild(nextBtn);
}

function renderCurrentPage() {
  const startIndex = (pagination.currentPage - 1) * pagination.pageSize;
  const endIndex = startIndex + pagination.pageSize;
  const pageTrades = allTrades.slice(startIndex, endIndex);

  renderTradesTable(pageTrades);
  updatePaginationInfo();
  updatePaginationButtons();
}

// Function to update analytics
async function updateAnalytics(query) {
  try {
    const a = await Journal.GetAnalytics(query);
    setMetric("winRate", (a.winRate * 100).toFixed(1) + "%");
    setMetric("profitFactor", a.profitFactor?.toFixed(2));
    setMetric("maxDD", a.maxDD?.toFixed(2) + "%");
    setMetric("sharpe", a.sharpe?.toFixed(3));
    setMetric("sortino", a.sortino?.toFixed(3));
    setMetric("expectancy", a.expectancy?.toFixed(2));
  } catch (e) {
    console.error("Failed to update analytics:", e);
    // Reset metrics on error
    setMetric("winRate", "Error");
    setMetric("profitFactor", "Error");
    setMetric("maxDD", "Error");
    setMetric("sharpe", "Error");
    setMetric("sortino", "Error");
    setMetric("expectancy", "Error");
  }
}

// Function to update trades with pagination
async function updateTradesTable(query) {
  try {
    allTrades = await Journal.ListTrades(query);
    pagination.totalTrades = allTrades.length;
    pagination.totalPages = Math.ceil(allTrades.length / pagination.pageSize);
    pagination.currentPage = 1; // Reset to first page

    renderCurrentPage();
  } catch (e) {
    console.error("Failed to load trades:", e);
    const tbody = document.getElementById("trades-tbody");
    if (tbody) {
      tbody.innerHTML =
        '<tr><td colspan="10" style="text-align: center; color: #fc8181;">Error loading trades</td></tr>';
    }
  }
}

// Function to render trades table
function renderTradesTable(trades) {
  const tbody = document.getElementById("trades-tbody");
  if (!tbody) return;

  if (!trades || trades.length === 0) {
    tbody.innerHTML = `
      <tr>
        <td colspan="10" style="text-align: center; color: #718096; padding: 40px;">
          No trades found for the current filters
        </td>
      </tr>
    `;
    return;
  }

  tbody.innerHTML = trades
    .map((trade) => {
      // Calculate P&L for closed trades
      let pnl = "";
      let pnlClass = "";

      if (trade.exit_price && trade.exit_price > 0) {
        const entryPrice = trade.entry_price;
        const exitPrice = trade.exit_price;
        const qty = trade.quantity;
        const fees = trade.fees || 0;

        let pnlValue;
        if (trade.side.toLowerCase() === "short") {
          pnlValue = (entryPrice - exitPrice) * qty - fees;
        } else {
          pnlValue = (exitPrice - entryPrice) * qty - fees;
        }

        pnl = "$" + pnlValue.toFixed(2);
        pnlClass = pnlValue >= 0 ? "pnl-positive" : "pnl-negative";
      } else {
        pnl = "-";
      }

      // Format dates
      const entryTime = formatDateTime(trade.entry_time);
      const exitTime = trade.exit_time
        ? formatDateTime(trade.exit_time)
        : "Open";

      // Format side with color
      const sideClass =
        trade.side.toLowerCase() === "long" ? "side-long" : "side-short";

      return `
        <tr>
          <td><strong>${escapeHtml(trade.symbol)}</strong></td>
          <td><span class="${sideClass}">${escapeHtml(
        trade.side.toUpperCase()
      )}</span></td>
          <td>${entryTime}</td>
          <td>${exitTime}</td>
          <td>$${trade.entry_price.toFixed(2)}</td>
          <td>${trade.exit_price ? "$" + trade.exit_price.toFixed(2) : "-"}</td>
          <td>${trade.quantity.toFixed(2)}</td>
          <td>$${trade.fees ? trade.fees.toFixed(2) : "0.00"}</td>
          <td class="${pnlClass}">${pnl}</td>
          <td>${escapeHtml(trade.notes || "")}</td>
        </tr>
      `;
    })
    .join("");
}

// Helper function to format datetime
function formatDateTime(dateStr) {
  if (!dateStr) return "-";
  try {
    const date = new Date(dateStr);
    return date.toLocaleString("en-US", {
      year: "numeric",
      month: "2-digit",
      day: "2-digit",
      hour: "2-digit",
      minute: "2-digit",
      timeZoneName: "short",
    });
  } catch (e) {
    return dateStr;
  }
}

// Helper function to escape HTML
function escapeHtml(text) {
  const div = document.createElement("div");
  div.textContent = text;
  return div.innerHTML;
}

// Apply filters function
async function applyFilters() {
  console.log("Applying filters...");
  const query = buildQueryFromFilters();

  try {
    // Update analytics
    await updateAnalytics(query);

    // Update trades table with pagination
    await updateTradesTable(query);

    // Update equity chart
    await updateEquityChart(query);
  } catch (error) {
    console.error("Error applying filters:", error);
  }
}

// Wire Apply Filters button
const applyBtn = document.getElementById("apply-filters");
if (applyBtn) {
  applyBtn.addEventListener("click", applyFilters);
}

// Page size change handler
const pageSizeSelect = document.getElementById("page-size");
if (pageSizeSelect) {
  pageSizeSelect.addEventListener("change", (e) => {
    pagination.pageSize = parseInt(e.target.value);
    pagination.totalPages = Math.ceil(
      pagination.totalTrades / pagination.pageSize
    );
    pagination.currentPage = 1; // Reset to first page
    renderCurrentPage();
  });
}

function setMetric(name, value) {
  const el = document.getElementById(`metric-${name}`);
  if (el) el.innerText = value ?? "-";
}

// Initialize the equity chart
function initEquityChart() {
  console.log("Initializing equity chart...");

  const canvas = document.getElementById("equityChart");
  if (!canvas) {
    console.error("Equity chart canvas element not found!");
    return;
  }

  if (typeof Chart === "undefined") {
    console.error("Chart.js is not loaded!");
    return;
  }

  try {
    const ctx = canvas.getContext("2d");
    equityChart = new Chart(ctx, {
      type: "line",
      data: {
        labels: ["No trades yet"],
        datasets: [
          {
            label: "Cumulative P&L",
            data: [0],
            borderColor: "#667eea",
            backgroundColor: "rgba(102, 126, 234, 0.1)",
            tension: 0.4,
            pointRadius: 4,
            pointHoverRadius: 6,
            pointBackgroundColor: "#667eea",
            pointBorderColor: "#ffffff",
            pointBorderWidth: 2,
            fill: true,
          },
        ],
      },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        plugins: {
          legend: {
            display: true,
          },
          tooltip: {
            backgroundColor: "rgba(0, 0, 0, 0.8)",
            titleColor: "#ffffff",
            bodyColor: "#ffffff",
            cornerRadius: 12,
            displayColors: false,
            callbacks: {
              label: function (context) {
                return `P&L: $${context.parsed.y.toFixed(2)}`;
              },
            },
          },
        },
        scales: {
          x: {
            display: true,
            title: {
              display: true,
              text: "Trade Sequence",
              color: "#4a5568",
              font: { size: 14, weight: "600" },
            },
            grid: { color: "rgba(255, 255, 255, 0.3)" },
          },
          y: {
            display: true,
            title: {
              display: true,
              text: "Cumulative P&L ($)",
              color: "#4a5568",
              font: { size: 14, weight: "600" },
            },
            grid: { color: "rgba(255, 255, 255, 0.3)" },
            ticks: {
              callback: function (value) {
                return "$" + value.toFixed(2);
              },
            },
          },
        },
        interaction: {
          intersect: false,
          mode: "index",
        },
      },
    });

    console.log("Equity chart initialized successfully");
  } catch (error) {
    console.error("Error initializing equity chart:", error);
  }
}

// Update equity chart with new data
async function updateEquityChart(query) {
  console.log("Updating equity chart with query:", query);

  if (!equityChart) {
    console.error("Equity chart not initialized!");
    return;
  }

  try {
    const equityPoints = await Journal.GetEquityPoints(query);
    console.log("Received equity points:", equityPoints);

    if (!equityPoints || equityPoints.length === 0) {
      // Show empty state
      equityChart.data.labels = ["No closed trades"];
      equityChart.data.datasets[0].data = [0];
      equityChart.update();
      return;
    }

    // Process the data
    const labels = [];
    const data = [];

    equityPoints.forEach((point, index) => {
      const date = new Date(point.t);
      const label = `${index + 1}: ${date.toLocaleDateString()}`;
      labels.push(label);
      data.push(point.v);
    });

    // Update chart
    equityChart.data.labels = labels;
    equityChart.data.datasets[0].data = data;

    // Color coding based on final P&L
    const finalPnL = data[data.length - 1] || 0;
    if (finalPnL >= 0) {
      equityChart.data.datasets[0].borderColor = "#38a169";
      equityChart.data.datasets[0].backgroundColor = "rgba(56, 161, 105, 0.1)";
      equityChart.data.datasets[0].pointBackgroundColor = "#38a169";
    } else {
      equityChart.data.datasets[0].borderColor = "#e53e3e";
      equityChart.data.datasets[0].backgroundColor = "rgba(229, 62, 62, 0.1)";
      equityChart.data.datasets[0].pointBackgroundColor = "#e53e3e";
    }

    equityChart.update();
    console.log("Chart updated successfully");
  } catch (error) {
    console.error("Error updating equity chart:", error);

    // Show error state
    equityChart.data.labels = ["Error loading data"];
    equityChart.data.datasets[0].data = [0];
    equityChart.update();
  }
}

// CSV Import via textarea (Mac compatible)
const importBtn = document.getElementById("import-btn");
const importRes = document.getElementById("import-result");

if (importBtn && importRes) {
  importBtn.addEventListener("click", async () => {
    const csvText = document.getElementById("csv-text")?.value || "";
    if (!csvText.trim()) {
      showImportResult("Please paste CSV data first", "error");
      return;
    }

    // Show loading state
    importRes.style.display = "block";
    importRes.className = "status";
    importRes.textContent = "Processing CSV data...";

    try {
      const report = await Journal.ImportCSV(csvText);

      const message = `Successfully imported ${
        report.imported
      } trades. Skipped: ${report.skipped}, Errors: ${
        report.errors?.length || 0
      }`;
      showImportResult(
        message,
        report.errors?.length > 0 ? "error" : "success"
      );

      // Clear textarea after successful import
      const textArea = document.getElementById("csv-text");
      if (textArea) textArea.value = "";

      // Refresh data after import
      await applyFilters();
    } catch (err) {
      showImportResult("Import failed: " + err.message, "error");
    }
  });
}

function showImportResult(message, type) {
  const resultDiv = document.getElementById("import-result");
  if (!resultDiv) return;

  resultDiv.textContent = message;
  resultDiv.className = `status ${type}`;
  resultDiv.style.display = "block";

  // Hide after 5 seconds
  setTimeout(() => {
    resultDiv.style.display = "none";
  }, 5000);
}

// Handle time events
Events.On("time", (time) => {
  if (timeElement) {
    timeElement.innerText = time.data;
  }
});

// Initialize everything on page load
document.addEventListener("DOMContentLoaded", () => {
  console.log("DOM Content Loaded - Starting initialization...");

  setTimeout(() => {
    // Initialize chart
    initEquityChart();

    // Set default date range to last 30 days
    const endDate = new Date();
    const startDate = new Date();
    startDate.setDate(startDate.getDate() - 30);

    const startInput = document.getElementById("filter-start");
    const endInput = document.getElementById("filter-end");

    if (startInput) startInput.value = startDate.toISOString().split("T")[0];
    if (endInput) endInput.value = endDate.toISOString().split("T")[0];

    // Test connection and load initial data
    setTimeout(async () => {
      console.log("Testing Journal service connection...");
      try {
        if (typeof Journal !== "undefined" && Journal.Ping) {
          const version = await Journal.Ping();
          console.log("Journal service connected, version:", version);

          // Auto-load data
          await applyFilters();
        } else {
          console.error("Journal service not available");
        }
      } catch (error) {
        console.error("Journal service test failed:", error);
      }
    }, 1000);
  }, 500);
});

// Handle window resize for chart
window.addEventListener("resize", () => {
  if (equityChart) {
    equityChart.resize();
  }
});
