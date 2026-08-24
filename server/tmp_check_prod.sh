#!/bin/bash
# 查 vOPC 项目现状（生产 MySQL）
DB_PW=$(grep '^DB_PASSWORD=' /etc/wxx/env | cut -d= -f2-)
DB_NAME=$(grep '^DB_NAME=' /etc/wxx/env | cut -d= -f2-)
DB_USER=$(grep '^DB_USER=' /etc/wxx/env | cut -d= -f2-)
: "${DB_USER:=wxx}"

echo "=== vopc 表是否存在 ==="
mysql --batch --skip-column-names -h127.0.0.1 -u"$DB_USER" -p"$DB_PW" "$DB_NAME" \
  -e "SELECT table_name FROM information_schema.tables WHERE table_schema='$DB_NAME' AND table_name LIKE 'vopc_%' ORDER BY table_name;" 2>/dev/null

echo "=== vOPC 项目列表 ==="
mysql --batch --skip-column-names -h127.0.0.1 -u"$DB_USER" -p"$DB_PW" "$DB_NAME" \
  -e "SELECT id,name,stage,status, IFNULL(product_form,'<EMPTY>') AS pf, IFNULL(project_cycle,'<EMPTY>') AS cycle FROM vopc_projects ORDER BY id DESC LIMIT 15;" 2>/dev/null

echo "=== 关键必填字段缺失统计（全部项目） ==="
mysql --batch --skip-column-names -h127.0.0.1 -u"$DB_USER" -p"$DB_PW" "$DB_NAME" \
  -e "SELECT 
     SUM(CASE WHEN product_form IS NULL OR product_form='' THEN 1 ELSE 0 END) AS missing_product_form,
     SUM(CASE WHEN project_cycle IS NULL OR project_cycle='' THEN 1 ELSE 0 END) AS missing_cycle,
     SUM(CASE WHEN acceptance_criteria IS NULL OR acceptance_criteria='' THEN 1 ELSE 0 END) AS missing_acceptance
     FROM vopc_projects;" 2>/dev/null
