import { Events } from "@wailsio/runtime";
import * as Journal from "./bindings/changeme/backend/bindings/journalservice.js";
import * as Models from "./bindings/changeme/backend/models/models.js";

const timeElement = document.getElementById("time");
let equityChart = null;

// Wire Ping button

const pingBtn = document.getElementById("ping-btn");
const pingResult = document.getElementById("ping-result");
if (pingBtn && pingResult) {
  pingBtn.addEventListener("click", async () => {
    try {
      const v = await Journal.Ping();
      pingResult.innerText = `version: ${v}`;
    } catch (e) {
      pingResult.innerText = "ping failed";
    }
  });
}

// Global query state
let currentQuery = Models.Query.createFrom({
  symbol: "",
  side: "",
  startTime: undefined,
  endTime: undefined,
  limit: 1000,
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
    limit: 1000,
    offset: 0,
  });

  return currentQuery;
}

// Function to update analytics
async function updateAnalytics(query) {
  try {
    const a = await Journal.GetAnalytics(query);
    setMetric("winRate", (a.winRate * 100).toFixed(1) + "%");
    setMetric("profitFactor", a.profitFactor?.toFixed(2));
    setMetric("maxDD", a.maxDD?.toFixed(2) + "%");
    setMetric("sharpe", a.sharpe?.toFixed(2));
    setMetric("sortino", a.sortino?.toFixed(2));
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

// Function to update trades table
async function updateTradesTable(query) {
  try {
    const trades = await Journal.ListTrades(query);
    renderTradesTable(trades);
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
    tbody.innerHTML =
      '<tr><td colspan="10" style="text-align: center; color: #a0aec0;">No trades found</td></tr>';
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

        pnl = pnlValue.toFixed(2);
        pnlClass = pnlValue >= 0 ? "pnl-positive" : "pnl-negative";
      } else {
        pnl = "-";
      }

      // Format dates
      const entryTime = formatDateTime(trade.entry_time);
      const exitTime = trade.exit_time ? formatDateTime(trade.exit_time) : "-";

      // Format side with color
      const sideClass =
        trade.side.toLowerCase() === "long" ? "side-long" : "side-short";

      return `
            <tr>
                <td><strong>${escapeHtml(trade.symbol)}</strong></td>
                <td><span class="${sideClass}">${escapeHtml(
        trade.side.toUpperCase()
      )}</span></td>
                <td class="datetime">${entryTime}</td>
                <td class="datetime">${exitTime}</td>
                <td class="number">${trade.entry_price.toFixed(2)}</td>
                <td class="number">${
                  trade.exit_price ? trade.exit_price.toFixed(2) : "-"
                }</td>
                <td class="number">${trade.quantity.toFixed(2)}</td>
                <td class="number">${
                  trade.fees ? trade.fees.toFixed(2) : "0.00"
                }</td>
                <td class="number ${pnlClass}">${pnl}</td>
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

// Filters → Query → GetAnalytics + ListTrades
const applyBtn = document.getElementById("apply-filters");
if (applyBtn) {
  // Remove existing listeners
  applyBtn.replaceWith(applyBtn.cloneNode(true));
  const newApplyBtn = document.getElementById("apply-filters");

  newApplyBtn.addEventListener("click", async () => {
    console.log("🔘 Apply Filters button clicked");
    await debugApplyFilters();
  });
}

// Refresh trades button
const refreshTradesBtn = document.getElementById("refresh-trades");
if (refreshTradesBtn) {
  refreshTradesBtn.addEventListener("click", async () => {
    await updateTradesTable(currentQuery);
  });
}

function setMetric(name, value) {
  const el = document.getElementById(`metric-${name}`);
  if (el) el.innerText = value ?? "-";
}

// Initialize the equity chart (add this to your DOMContentLoaded event)
function initEquityChart() {
  console.log(" Initializing equity chart...");

  const canvas = document.getElementById("equityChart");
  if (!canvas) {
    console.error(" Equity chart canvas element not found!");
    return;
  }

  console.log(" Found equity chart canvas element");

  // Check if Chart.js is loaded
  if (typeof Chart === "undefined") {
    console.error(" Chart.js is not loaded!");
    return;
  }

  console.log(" Chart.js is loaded");

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
            borderColor: "#007acc",
            backgroundColor: "rgba(0, 122, 204, 0.05)",
            tension: 0.2,
            pointRadius: 4,
            pointHoverRadius: 6,
            pointBackgroundColor: "#007acc",
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
            cornerRadius: 8,
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
              font: {
                size: 14,
                weight: "bold",
              },
            },
            grid: {
              color: "rgba(0, 0, 0, 0.1)",
            },
          },
          y: {
            display: true,
            title: {
              display: true,
              text: "Cumulative P&L ($)",
              font: {
                size: 14,
                weight: "bold",
              },
            },
            grid: {
              color: "rgba(0, 0, 0, 0.1)",
            },
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

    console.log(" Equity chart initialized successfully");
  } catch (error) {
    console.error(" Error initializing equity chart:", error);
  }
}

// Update equity chart with new data
async function updateEquityChart(query) {
  console.log(" Updating equity chart with query:", query);

  if (!equityChart) {
    console.error(" Equity chart not initialized!");
    return;
  }

  try {
    console.log(" Calling Journal.GetEquityPoints...");

    // Check if Journal is available
    if (typeof Journal === "undefined") {
      console.error(" Journal service not available!");
      return;
    }

    const equityPoints = await Journal.GetEquityPoints(query);
    console.log(" Received equity points:", equityPoints);

    if (!equityPoints) {
      console.warn(" No equity points returned (null/undefined)");
      // Show empty state
      equityChart.data.labels = ["No data"];
      equityChart.data.datasets[0].data = [0];
      equityChart.update();
      return;
    }

    if (equityPoints.length === 0) {
      console.warn(" Empty equity points array");
      // Show empty state
      equityChart.data.labels = ["No closed trades"];
      equityChart.data.datasets[0].data = [0];
      equityChart.update();
      return;
    }

    console.log(` Processing ${equityPoints.length} equity points`);

    // Process the data
    const labels = [];
    const data = [];

    equityPoints.forEach((point, index) => {
      console.log(`Point ${index}:`, { time: point.t, value: point.v });

      const date = new Date(point.t);
      const label = `${index + 1}: ${date.toLocaleDateString()}`;

      labels.push(label);
      data.push(point.v);
    });

    console.log(" Chart labels:", labels);
    console.log(" Chart data:", data);

    // Update chart
    equityChart.data.labels = labels;
    equityChart.data.datasets[0].data = data;

    // Color coding based on final P&L
    const finalPnL = data[data.length - 1] || 0;
    console.log(` Final P&L: $${finalPnL}`);

    if (finalPnL >= 0) {
      equityChart.data.datasets[0].borderColor = "#28a745";
      equityChart.data.datasets[0].backgroundColor = "rgba(40, 167, 69, 0.1)";
      equityChart.data.datasets[0].pointBackgroundColor = "#28a745";
      console.log(" Chart colored green (profitable)");
    } else {
      equityChart.data.datasets[0].borderColor = "#dc3545";
      equityChart.data.datasets[0].backgroundColor = "rgba(220, 53, 69, 0.1)";
      equityChart.data.datasets[0].pointBackgroundColor = "#dc3545";
      console.log(" Chart colored red (loss)");
    }

    equityChart.update();
    console.log(" Chart updated successfully");
  } catch (error) {
    console.error(" Error updating equity chart:", error);
    console.error("Error details:", error.message, error.stack);

    // Show error state
    equityChart.data.labels = ["Error loading data"];
    equityChart.data.datasets[0].data = [0];
    equityChart.update();
  }
}

async function testEquityPoints() {
  console.log("🧪 Testing equity points manually...");

  try {
    // Test with empty query
    const emptyQuery = {
      symbol: "",
      side: "",
      startTime: undefined,
      endTime: undefined,
      limit: 1000,
      offset: 0,
    };

    console.log(" Testing with empty query:", emptyQuery);
    const points = await Journal.GetEquityPoints(emptyQuery);
    console.log(" Test result:", points);

    return points;
  } catch (error) {
    console.error(" Test failed:", error);
    return null;
  }
}

async function debugApplyFilters() {
  console.log(" Debug: Applying filters...");

  const query = buildQueryFromFilters();
  console.log(" Built query:", query);

  try {
    // Test equity points first
    await testEquityPoints();

    // Update chart
    await updateEquityChart(query);

    // Also update other components
    await updateAnalytics(query);
    await updateTradesTable(query);
  } catch (error) {
    console.error(" Debug apply filters failed:", error);
  }
}

// CSV Import with improved feedback
const importBtn = document.getElementById("import-btn");
const importRes = document.getElementById("import-result");
if (importBtn && importRes) {
  importBtn.addEventListener("click", async () => {
    const txt = document.getElementById("csv-text")?.value || "";

    if (!txt.trim()) {
      importRes.innerHTML =
        '<span style="color: #fc8181;">Please paste CSV data</span>';
      return;
    }

    importRes.innerHTML = '<span style="color: #90cdf4;">Importing...</span>';

    try {
      const report = await Journal.ImportCSV(txt);

      let resultHtml = `<strong>Import Complete:</strong><br>`;
      resultHtml += ` Imported: ${report.imported}<br>`;
      resultHtml += ` Skipped (duplicates): ${report.skipped}<br>`;

      if (report.errors && report.errors.length > 0) {
        resultHtml += ` Errors: ${report.errors.length}<br>`;
        resultHtml += `<details style="margin-top: 8px;"><summary>Show Errors</summary>`;
        resultHtml += `<div style="font-family: monospace; font-size: 0.8rem; margin-top: 4px;">`;
        report.errors.forEach((error) => {
          resultHtml += `• ${escapeHtml(error)}<br>`;
        });
        resultHtml += `</div></details>`;
      }

      importRes.innerHTML = resultHtml;

      // If import was successful, refresh the current view
      if (report.imported > 0) {
        setTimeout(async () => {
          await Promise.all([
            updateAnalytics(currentQuery),
            updateTradesTable(currentQuery),
          ]);
        }, 500);
      }
    } catch (e) {
      importRes.innerHTML = `<span style="color: #fc8181;">Import failed: ${escapeHtml(
        e.message || "Unknown error"
      )}</span>`;
    }
  });
}

// Load initial empty state
Events.On("time", (time) => {
  if (timeElement) {
    timeElement.innerText = time.data;
  }
});

// Auto-load trades on page load with empty filters
document.addEventListener("DOMContentLoaded", () => {
  console.log(" DOM Content Loaded - Starting initialization...");

  // Wait a bit for everything to load
  setTimeout(() => {
    console.log(" Delayed initialization starting...");

    // Initialize chart
    initEquityChart();

    // Test the connection
    setTimeout(async () => {
      console.log(" Testing Journal service connection...");
      try {
        if (typeof Journal !== "undefined" && Journal.Ping) {
          const version = await Journal.Ping();
          console.log(" Journal service connected, version:", version);

          // Test equity points
          await testEquityPoints();
        } else {
          console.error(" Journal service not available");
        }
      } catch (error) {
        console.error(" Journal service test failed:", error);
      }
    }, 1000);
  }, 500);
});
