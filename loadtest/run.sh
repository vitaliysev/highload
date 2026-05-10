#!/bin/bash
SCRIPT="${1:-smoke}"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
REPORT="loadtest/reports/${SCRIPT}_${TIMESTAMP}.html"

mkdir -p loadtest/reports

K6_WEB_DASHBOARD=true \
K6_WEB_DASHBOARD_EXPORT="$REPORT" \
k6 run "loadtest/${SCRIPT}.js"

echo ""
echo "Отчёт сохранён: $REPORT"
