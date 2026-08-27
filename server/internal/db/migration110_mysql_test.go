package db

import (
	"strings"
	"testing"
)

// TestToMySQL_InsertSelectWhereNotExists 验证迁移 110 的 INSERT...SELECT...WHERE NOT EXISTS
// 在 MySQL 方言转换下是否产生合法 SQL。
func TestToMySQL_InsertSelectWhereNotExists(t *testing.T) {
	sql := `INSERT INTO campus_checkin_steps
    (campus_id,step_order,title,location,lat,lng,duration,task,materials,contact,note,icon_name,status)
SELECT 'huifeng',1,'校门入校核验','会峰校区南门',32.2705,118.3055,'约 5 分钟','x','y','z','n','login','published'
WHERE NOT EXISTS (SELECT 1 FROM campus_checkin_steps WHERE campus_id='huifeng' AND step_order=1 AND status='published');`

	out := ToMySQL(sql)
	t.Logf("转换结果:\n%s", out)

	if out == "" {
		t.Fatal("ToMySQL 返回空串")
	}
	// 关键断言：不能把 NOT EXISTS 或 WHERE 子句弄丢
	upper := strings.ToUpper(out)
	for _, must := range []string{"INSERT INTO", "SELECT", "WHERE NOT EXISTS", "CAMPUS_CHECKIN_STEPS"} {
		if !strings.Contains(upper, must) {
			t.Errorf("转换后缺少关键片段 %q:\n%s", must, out)
		}
	}
}
