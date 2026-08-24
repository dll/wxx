#!/bin/bash
# 补全「测试」项目的必填占位字段（字段待定，先用占位值），使提交立项能通过。
# 仅针对 id 对应 name='测试' 且仍为 S0/draft 的项目，避免误改已推进项目。
DB_PW=$(grep '^DB_PASSWORD=' /etc/wxx/env | cut -d= -f2-)
DB_NAME=$(grep '^DB_NAME=' /etc/wxx/env | cut -d= -f2-)
DB_USER=$(grep '^DB_USER=' /etc/wxx/env | cut -d= -f2-)
: "${DB_USER:=wxx}"

PID=$(mysql --batch --skip-column-names -h127.0.0.1 -u"$DB_USER" -p"$DB_PW" "$DB_NAME" \
  -e "SELECT id FROM vopc_projects WHERE name='测试' AND stage='S0' AND status='draft' ORDER BY id LIMIT 1;" 2>/dev/null)

echo "TARGET_PROJECT_ID=$PID"
if [ -z "$PID" ]; then echo "NO_TARGET"; exit 0; fi

mysql -h127.0.0.1 -u"$DB_USER" -p"$DB_PW" "$DB_NAME" <<SQL 2>/dev/null
UPDATE vopc_projects SET
  product_form=COALESCE(NULLIF(product_form,''),'Web 应用'),
  project_cycle=COALESCE(NULLIF(project_cycle,''),'8 周'),
  acceptance_criteria=COALESCE(NULLIF(acceptance_criteria,''),'功能可用、验收通过'),
  summary=COALESCE(NULLIF(summary,''),'测试项目摘要'),
  problem_statement=COALESCE(NULLIF(problem_statement,''),'待解决问题'),
  target_users=COALESCE(NULLIF(target_users,''),'目标用户'),
  expected_outcome=COALESCE(NULLIF(expected_outcome,''),'预期成果'),
  validation_plan=COALESCE(NULLIF(validation_plan,''),'用户验证'),
  updated_at=CURRENT_TIMESTAMP
WHERE id=$PID AND stage='S0' AND status='draft';
SQL
echo "UPDATE_EXIT=$?"

echo "=== 补全后检查 ==="
mysql --batch --skip-column-names -h127.0.0.1 -u"$DB_USER" -p"$DB_PW" "$DB_NAME" \
  -e "SELECT id,name,product_form,project_cycle,acceptance_criteria,summary FROM vopc_projects WHERE id=$PID;" 2>/dev/null
