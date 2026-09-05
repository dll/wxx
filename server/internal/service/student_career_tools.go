package service

import (
	"context"
	"fmt"
	"strings"
)

type ResumeData struct {
	Title      string                   `json:"title"`
	Template   string                   `json:"template"`
	Sections   []map[string]interface{} `json:"sections"`
	DataSource string                   `json:"data_source"`
}

func (s *StudentService) GenerateResume(ctx context.Context, userID int64, position string) *ResumeData {
	_ = ctx
	userName, major, college, eduLine, dataSource := "同学", "", "", "请补充学校、专业与起止年份", "fallback"
	if s.userRepo != nil {
		if user, err := s.userRepo.GetByID(userID); err == nil && user != nil {
			if user.DisplayName != "" {
				userName = user.DisplayName
			}
			major, college = user.Major, user.College
			parts := make([]string, 0, 4)
			if college != "" {
				parts = append(parts, college)
			}
			if major != "" {
				parts = append(parts, major)
			}
			if user.EnrollmentYear != "" {
				parts = append(parts, user.EnrollmentYear+"级")
			}
			if len(parts) > 0 {
				eduLine, dataSource = strings.Join(parts, " · "), "real"
			}
		}
	}
	if position == "" {
		position = "（请填写目标岗位）"
	}
	info := []string{userName}
	if major != "" {
		info = append(info, major)
	}
	return &ResumeData{Title: userName + "的简历 - 应聘" + position, Template: "现代简洁风", Sections: []map[string]interface{}{{"section": "个人信息", "content": strings.Join(info, " | "), "editable": true, "hint": "补充联系方式、政治面貌、外语水平等"}, {"section": "教育背景", "content": eduLine, "editable": true, "hint": "确认学校、专业、起止年份与 GPA"}, {"section": "项目经历", "content": "", "editable": true, "hint": "按 项目名称 + 技术栈 + 你的职责与成果 逐条填写真实项目"}, {"section": "技能特长", "content": "", "editable": true, "hint": "列出你实际掌握的语言、框架与工具"}, {"section": "获奖经历", "content": "", "editable": true, "hint": "填写真实获得的奖项与荣誉，切勿虚构"}}, DataSource: dataSource}
}

type CareerSimulation struct {
	CareerPath  string                   `json:"career_path"`
	ThreeYear   string                   `json:"three_year"`
	FiveYear    string                   `json:"five_year"`
	Skills      []string                 `json:"skills_needed"`
	SalaryTrend []map[string]interface{} `json:"salary_trend"`
	DataSource  string                   `json:"data_source"`
}

func (s *StudentService) GenerateCareerSimulation(ctx context.Context, careerPath string) *CareerSimulation {
	_ = ctx
	if careerPath == "" {
		careerPath = "后端开发工程师"
	}
	return &CareerSimulation{CareerPath: careerPath, ThreeYear: fmt.Sprintf("3年后的你：成为%s技术骨干，独立负责核心模块开发，年薪20-30万", careerPath), FiveYear: "5年后的你：成为技术专家或Team Leader，带领3-5人团队，年薪30-50万", Skills: []string{"Java/Python/Go精通", "分布式系统设计", "技术团队管理", "业务架构能力"}, SalaryTrend: []map[string]interface{}{{"year": "2026(应届)", "range": "8-12万", "title": "初级开发工程师"}, {"year": "2028(3年)", "range": "20-30万", "title": "高级开发工程师"}, {"year": "2030(5年)", "range": "30-50万", "title": "技术专家/架构师"}}, DataSource: "reference"}
}

type AlumniMatch struct {
	Matches          []map[string]interface{} `json:"matches"`
	SuggestQuestions []string                 `json:"suggest_questions"`
	DataSource       string                   `json:"data_source"`
}

func (s *StudentService) GenerateAlumniMatch(ctx context.Context, userID int64) *AlumniMatch {
	_ = ctx
	return &AlumniMatch{Matches: []map[string]interface{}{{"name": "陈学长", "grad_year": "2023", "company": "字节跳动", "position": "后端开发", "similarity": "兴趣方向高度匹配", "match_score": 92}, {"name": "刘学姐", "grad_year": "2022", "company": "阿里巴巴", "position": "Java开发", "similarity": "技术栈相似", "match_score": 85}, {"name": "王学长", "grad_year": "2024", "company": "美团", "position": "Golang开发", "similarity": "项目经历相近", "match_score": 78}}, SuggestQuestions: []string{"您觉得大学期间最值得投入时间学习的技术是什么？", "实习面试时面试官最看重哪些能力？", "从学校到职场的转变中，最大的挑战是什么？"}, DataSource: "reference"}
}
