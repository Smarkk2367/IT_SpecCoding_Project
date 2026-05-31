package report

import (
	"fmt"
	"html/template"
	"strings"
	"time"
)

const ReportHTMLTemplate = `<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <title>TrackFlow Report</title>
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&family=Outfit:wght@500;600;700;800&display=swap" rel="stylesheet">
    <style>
        :root {
            --primary: #0f172a;
            --accent: #6366f1;
            --accent-dark: #4f46e5;
            --bg: #f8fafc;
            --card-bg: #ffffff;
            --border: #e2e8f0;
            --text-main: #334155;
            --text-dark: #0f172a;
            --text-muted: #64748b;
        }
        
        * {
            box-sizing: border-box;
            margin: 0;
            padding: 0;
        }

        body {
            font-family: 'Inter', sans-serif;
            background-color: var(--bg);
            color: var(--text-main);
            line-height: 1.5;
            padding: 40px;
            -webkit-print-color-adjust: exact;
        }

        .header {
            display: flex;
            justify-content: space-between;
            align-items: flex-start;
            border-bottom: 2px solid var(--border);
            padding-bottom: 24px;
            margin-bottom: 30px;
        }

        .logo-section h1 {
            font-family: 'Outfit', sans-serif;
            font-weight: 800;
            font-size: 28px;
            color: var(--accent);
            letter-spacing: -0.5px;
            margin-bottom: 4px;
        }

        .logo-section p {
            font-size: 14px;
            color: var(--text-muted);
            text-transform: uppercase;
            letter-spacing: 1px;
            font-weight: 600;
        }

        .meta-section {
            text-align: right;
        }

        .meta-section h2 {
            font-family: 'Outfit', sans-serif;
            font-weight: 700;
            font-size: 20px;
            color: var(--text-dark);
            margin-bottom: 6px;
        }

        .meta-section p {
            font-size: 13px;
            color: var(--text-muted);
        }

        .badge-date {
            display: inline-block;
            background-color: #e0e7ff;
            color: var(--accent-dark);
            padding: 6px 12px;
            border-radius: 9999px;
            font-size: 12px;
            font-weight: 600;
            margin-top: 8px;
        }

        .metrics-grid {
            display: grid;
            grid-template-columns: repeat(3, 1fr);
            gap: 20px;
            margin-bottom: 30px;
        }

        .metric-card {
            background-color: var(--card-bg);
            border: 1px solid var(--border);
            border-radius: 12px;
            padding: 20px;
            box-shadow: 0 1px 3px rgba(0,0,0,0.05);
            position: relative;
            overflow: hidden;
        }

        .metric-card::before {
            content: '';
            position: absolute;
            top: 0;
            left: 0;
            width: 4px;
            height: 100%;
            background-color: var(--accent);
        }

        .metric-card .label {
            font-size: 12px;
            font-weight: 600;
            text-transform: uppercase;
            color: var(--text-muted);
            letter-spacing: 0.5px;
            margin-bottom: 8px;
        }

        .metric-card .value {
            font-family: 'Outfit', sans-serif;
            font-weight: 700;
            font-size: 32px;
            color: var(--text-dark);
        }

        .section-title {
            font-family: 'Outfit', sans-serif;
            font-weight: 700;
            font-size: 18px;
            color: var(--text-dark);
            margin-bottom: 16px;
            display: flex;
            align-items: center;
            gap: 8px;
        }

        .section-title::after {
            content: '';
            flex: 1;
            height: 1px;
            background-color: var(--border);
        }

        .card {
            background-color: var(--card-bg);
            border: 1px solid var(--border);
            border-radius: 12px;
            padding: 24px;
            box-shadow: 0 1px 3px rgba(0,0,0,0.05);
            margin-bottom: 30px;
        }

        .chart-container {
            width: 100%;
            height: 220px;
            display: flex;
            justify-content: center;
            align-items: center;
        }

        .chart-svg {
            width: 100%;
            height: 100%;
        }

        .breakdown-grid {
            display: grid;
            grid-template-columns: repeat(3, 1fr);
            gap: 20px;
            margin-bottom: 30px;
        }

        .breakdown-card {
            background-color: var(--card-bg);
            border: 1px solid var(--border);
            border-radius: 12px;
            padding: 20px;
            box-shadow: 0 1px 3px rgba(0,0,0,0.05);
        }

        .breakdown-title {
            font-family: 'Outfit', sans-serif;
            font-weight: 600;
            font-size: 14px;
            color: var(--text-dark);
            margin-bottom: 14px;
            text-transform: uppercase;
            letter-spacing: 0.5px;
        }

        .bar-row {
            margin-bottom: 12px;
        }

        .bar-row:last-child {
            margin-bottom: 0;
        }

        .bar-labels {
            display: flex;
            justify-content: space-between;
            font-size: 12px;
            margin-bottom: 4px;
        }

        .bar-label {
            font-weight: 500;
            color: var(--text-main);
        }

        .bar-value {
            font-weight: 600;
            color: var(--text-dark);
        }

        .bar-track {
            height: 8px;
            background-color: #f1f5f9;
            border-radius: 4px;
            overflow: hidden;
        }

        .bar-fill {
            height: 100%;
            background-color: var(--accent);
            border-radius: 4px;
        }

        .table-container {
            width: 100%;
            overflow-x: auto;
        }

        table {
            width: 100%;
            border-collapse: collapse;
            text-align: left;
        }

        th {
            background-color: #f1f5f9;
            color: var(--text-dark);
            font-weight: 600;
            font-size: 12px;
            text-transform: uppercase;
            letter-spacing: 0.5px;
            padding: 10px 16px;
            border-bottom: 2px solid var(--border);
        }

        td {
            padding: 12px 16px;
            border-bottom: 1px solid var(--border);
            font-size: 13px;
        }

        tr:last-child td {
            border-bottom: none;
        }

        .short-code-cell {
            font-family: monospace;
            font-weight: 600;
            color: var(--accent-dark);
        }

        .campaign-cell {
            font-style: italic;
            color: var(--text-muted);
        }

        .footer {
            text-align: center;
            font-size: 11px;
            color: var(--text-muted);
            margin-top: 40px;
            border-top: 1px solid var(--border);
            padding-top: 16px;
        }

        @media print {
            body {
                padding: 0;
                background-color: #ffffff;
            }
            .card, .metric-card, .breakdown-card {
                box-shadow: none;
            }
            .page-break {
                page-break-before: always;
            }
        }
    </style>
</head>
<body>
    <div class="header">
        <div class="logo-section">
            <h1>TrackFlow</h1>
            <p>Campaign Performance</p>
        </div>
        <div class="meta-section">
            <h2>{{if .ClientName}}Client: {{.ClientName}}{{else}}General Report{{end}}</h2>
            <p>Report ID: {{.ReportID}}</p>
            <span class="badge-date">{{formatDate .DateFrom}} - {{formatDate .DateTo}}</span>
        </div>
    </div>

    <div class="metrics-grid">
        <div class="metric-card">
            <div class="label">Total Clicks</div>
            <div class="value">{{.TotalClicks}}</div>
        </div>
        <div class="metric-card">
            <div class="label">Unique Clicks</div>
            <div class="value">{{.UniqueClicks}}</div>
        </div>
        <div class="metric-card">
            <div class="label">Active Links</div>
            <div class="value">{{len .Links}}</div>
        </div>
    </div>

    <h3 class="section-title">Clicks Trend</h3>
    <div class="card">
        <div class="chart-container">
            {{renderChart .ClicksOverTime}}
        </div>
    </div>

    <div class="breakdown-grid">
        <div class="breakdown-card">
            <h4 class="breakdown-title">Top Countries</h4>
            {{$total := .TotalClicks}}
            {{range .TopCountries}}
            <div class="bar-row">
                <div class="bar-labels">
                    <span class="bar-label">{{.Country}}</span>
                    <span class="bar-value">{{.Count}}</span>
                </div>
                <div class="bar-track">
                    <div class="bar-fill" style="width: {{percentage .Count $total}}%"></div>
                </div>
            </div>
            {{else}}
            <p style="font-size: 12px; color: var(--text-muted);">No country data available</p>
            {{end}}
        </div>

        <div class="breakdown-card">
            <h4 class="breakdown-title">Top Devices</h4>
            {{range .TopDevices}}
            <div class="bar-row">
                <div class="bar-labels">
                    <span class="bar-label">{{.DeviceType}}</span>
                    <span class="bar-value">{{.Count}}</span>
                </div>
                <div class="bar-track">
                    <div class="bar-fill" style="width: {{percentage .Count $total}}%"></div>
                </div>
            </div>
            {{else}}
            <p style="font-size: 12px; color: var(--text-muted);">No device data available</p>
            {{end}}
        </div>

        <div class="breakdown-card">
            <h4 class="breakdown-title">Top Referrers</h4>
            {{range .TopReferrers}}
            <div class="bar-row">
                <div class="bar-labels">
                    <span class="bar-label">{{.Referrer}}</span>
                    <span class="bar-value">{{.Count}}</span>
                </div>
                <div class="bar-track">
                    <div class="bar-fill" style="width: {{percentage .Count $total}}%"></div>
                </div>
            </div>
            {{else}}
            <p style="font-size: 12px; color: var(--text-muted);">No referrer data available</p>
            {{end}}
        </div>
    </div>

    <div class="page-break"></div>

    <h3 class="section-title">Links Performance</h3>
    <div class="card" style="padding: 12px;">
        <div class="table-container">
            <table>
                <thead>
                    <tr>
                        <th>Short Code</th>
                        <th>Original URL</th>
                        <th>Campaign</th>
                        <th>Total Clicks</th>
                        <th>Unique Clicks</th>
                    </tr>
                </thead>
                <tbody>
                    {{range .Links}}
                    <tr>
                        <td class="short-code-cell">{{.ShortCode}}</td>
                        <td>{{.OriginalURL}}</td>
                        <td class="campaign-cell">{{if .CampaignName}}{{.CampaignName}}{{else}}-{{end}}</td>
                        <td><strong>{{.TotalClicks}}</strong></td>
                        <td>{{.UniqueClicks}}</td>
                    </tr>
                    {{end}}
                </tbody>
            </table>
        </div>
    </div>

    <div class="footer">
        Generated automatically by TrackFlow Campaigns Tracking System on {{nowFormatted}}
    </div>
</body>
</html>`

func GetTemplate() (*template.Template, error) {
	funcs := template.FuncMap{
		"formatDate": func(t time.Time) string {
			return t.Format("02.01.2006")
		},
		"nowFormatted": func() string {
			return time.Now().UTC().Format("02.01.2006 15:04:05 MST")
		},
		"percentage": func(count, total int) int {
			if total == 0 {
				return 0
			}
			return int(float64(count) / float64(total) * 100)
		},
		"renderChart": renderChart,
	}

	return template.New("report").Funcs(funcs).Parse(ReportHTMLTemplate)
}

func renderChart(clicks []TimeCount) template.HTML {
	if len(clicks) == 0 {
		return template.HTML("<div style='font-size: 13px; color: #64748b;'>No click trend data available</div>")
	}

	maxCount := 0
	for _, c := range clicks {
		if c.Count > maxCount {
			maxCount = c.Count
		}
	}
	if maxCount == 0 {
		maxCount = 1
	}

	width := 700
	height := 200
	paddingLeft := 40
	paddingRight := 20
	paddingTop := 20
	paddingBottom := 30
	chartWidth := width - paddingLeft - paddingRight
	chartHeight := height - paddingTop - paddingBottom

	barWidth := float64(chartWidth) / float64(len(clicks))
	barSpacing := barWidth * 0.2
	rectWidth := barWidth - barSpacing

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`<svg viewBox="0 0 %d %d" class="chart-svg" xmlns="http://www.w3.org/2000/svg">`, width, height))

	// Draw Grid lines
	for i := 0; i <= 4; i++ {
		yVal := paddingTop + chartHeight*i/4
		countVal := maxCount - maxCount*i/4
		sb.WriteString(fmt.Sprintf(`<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="#F1F5F9" stroke-width="1" />`, paddingLeft, yVal, width-paddingRight, yVal))
		sb.WriteString(fmt.Sprintf(`<text x="%d" y="%d" font-size="9" font-family="'Inter', sans-serif" fill="#94A3B8" text-anchor="end" alignment-baseline="middle">%d</text>`, paddingLeft-8, yVal, countVal))
	}

	// Draw bars
	for i, c := range clicks {
		x := float64(paddingLeft) + float64(i)*barWidth + barSpacing/2
		barHeight := float64(c.Count) / float64(maxCount) * float64(chartHeight)
		y := float64(paddingTop) + float64(chartHeight) - barHeight

		// Sleek gradient bars
		sb.WriteString(fmt.Sprintf(`<rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" rx="3" fill="url(#barGradient)" />`, x, y, rectWidth, barHeight))

		// Hover/value labels above the bars (if count > 0)
		if c.Count > 0 {
			sb.WriteString(fmt.Sprintf(`<text x="%.2f" y="%.2f" font-size="9" font-family="'Inter', sans-serif" font-weight="600" fill="#6366F1" text-anchor="middle">%d</text>`, x+rectWidth/2, y-6, c.Count))
		}

		// X-Axis Labels
		dateStr := c.Timestamp.Format("02.01")
		// Prevent overlapping labels
		step := 1
		if len(clicks) > 14 {
			step = len(clicks) / 7
		}
		if i%step == 0 {
			sb.WriteString(fmt.Sprintf(`<text x="%.2f" y="%d" font-size="9" font-family="'Inter', sans-serif" fill="#64748B" text-anchor="middle">%s</text>`, x+rectWidth/2, height-paddingBottom+16, dateStr))
		}
	}

	// Gradients definition
	sb.WriteString(`
		<defs>
			<linearGradient id="barGradient" x1="0%" y1="0%" x2="0%" y2="100%">
				<stop offset="0%" stop-color="#6366F1" />
				<stop offset="100%" stop-color="#4F46E5" />
			</linearGradient>
		</defs>
	`)
	sb.WriteString("</svg>")
	return template.HTML(sb.String())
}
