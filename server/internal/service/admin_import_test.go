package service

import (
	"os"
	"testing"

	"github.com/dll/wxx/server/internal/repository"
	"github.com/dll/wxx/server/internal/testutil"
	"golang.org/x/crypto/bcrypt"
)

func TestAdminService_ParseStudentXLSX_ReferenceFile(t *testing.T) {
	file, err := os.Open("../../../data/学生名单.xlsx")
	if os.IsNotExist(err) {
		t.Skip("本地未提供 data/学生名单.xlsx")
	}
	if err != nil {
		t.Fatalf("打开参考 Excel 失败: %v", err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		t.Fatalf("读取参考 Excel 信息失败: %v", err)
	}

	rows, err := (&AdminService{}).ParseStudentXLSX(file, info.Size())
	if err != nil {
		t.Fatalf("解析参考 Excel 失败: %v", err)
	}
	if len(rows) != 5 {
		t.Fatalf("参考 Excel 应包含 5 名学生，实际 %d", len(rows))
	}
	first := rows[0]
	if first.Username != "2023211981" || first.DisplayName != "张明远" ||
		first.College != "计算机学院" || first.ClassName != "软开23" ||
		first.EnrollmentYear != "2023" || first.Role != "学生" {
		t.Fatalf("参考 Excel 首行映射错误: %+v", first)
	}
}

func TestAdminService_ImportStudents_EncryptsPasswordAndStoresProfile(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()
	repo := repository.NewUserRepo(db)
	svc := &AdminService{userRepo: repo}

	result, err := svc.ImportStudents([]*ImportStudentRow{{
		Username:       "2023211981",
		DisplayName:    "张明远",
		College:        "计算机学院",
		Major:          "计算机科学与技术",
		ClassName:      "软开23",
		EnrollmentDate: "2023-09-04",
		EnrollmentYear: "2023",
		Role:           "学生",
	}}, "", "counselor", "college", "cs")
	if err != nil {
		t.Fatalf("导入学生失败: %v", err)
	}
	if result.Success != 1 || result.Failed != 0 {
		t.Fatalf("导入统计错误: %+v", result)
	}

	user, err := repo.GetByUsername("2023211981")
	if err != nil || user == nil {
		t.Fatalf("查询导入用户失败: user=%+v err=%v", user, err)
	}
	if user.OwnerID != "cs" || user.College != "计算机学院" || user.ClassName != "软开23" {
		t.Fatalf("学生归属或资料保存错误: %+v", user)
	}
	if user.PasswordHash == "2023211981" {
		t.Fatal("数据库不得保存明文密码")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte("2023211981")); err != nil {
		t.Fatalf("默认学号密码验证失败: %v", err)
	}
}

func TestAdminService_ImportStudents_RejectsNonStudentRole(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()
	svc := &AdminService{userRepo: repository.NewUserRepo(db)}

	result, err := svc.ImportStudents([]*ImportStudentRow{{
		Username: "T001", DisplayName: "错误角色", Role: "教师",
	}}, "123456", "sys_admin", "school", "all")
	if err != nil {
		t.Fatalf("行级校验不应中断整个导入: %v", err)
	}
	if result.Success != 0 || result.Failed != 1 || result.Details[0].Error == "" {
		t.Fatalf("非学生角色应导入失败: %+v", result)
	}
}

func TestAdminService_ImportStudents_RejectsShortSharedPassword(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()
	svc := &AdminService{userRepo: repository.NewUserRepo(db)}
	_, err := svc.ImportStudents([]*ImportStudentRow{{
		Username: "2023211981", DisplayName: "张明远",
	}}, "123", "counselor", "college", "cs")
	if err == nil {
		t.Fatal("过短统一密码应被拒绝")
	}
}
