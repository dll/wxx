package handler

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"sync"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/gin-gonic/gin"
)

// GET /api/v1/student/profile
// 并发聚合：基础信息 + 学业 + 竞赛 + 入党 + 社团 + 打卡 + 积分
// 错误容忍：单个子查询失败不影响整体，返回空数组/零值
// SQL 已下沉 StudentProfileRepo（P4-d）
func (h *StudentHandler) PersonalProfile(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未登录"})
		return
	}

	userID := userCtx.UserID
	result := gin.H{"user_id": userID, "username": userCtx.Username, "display_name": userCtx.DisplayName}
	var wg sync.WaitGroup
	var mu sync.Mutex

	// 用匿名函数并发查询，写回 result
	query := func(key string, fn func() (interface{}, error)) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			val, err := fn()
			if err != nil {
				log.Printf("聚合[%s]失败 user_id=%d: %v", key, userID, err)
				return
			}
			mu.Lock()
			result[key] = val
			mu.Unlock()
		}()
	}

	// 1. 基础信息（users 表）
	query("basic_info", func() (interface{}, error) {
		info, err := h.profileRepo.GetBasicInfo(userID)
		if err != nil {
			return nil, err
		}
		return gin.H{
			"college": info.College, "major": info.Major, "class_name": info.ClassName,
			"enrollment_date": info.EnrollmentDate, "enrollment_year": info.EnrollmentYear, "status": info.Status,
		}, nil
	})

	// 2. 学业成绩汇总（student_grades）
	query("academic", func() (interface{}, error) {
		s, err := h.profileRepo.GetAcademicSummary(userID)
		if err != nil {
			return nil, err
		}
		return gin.H{
			"course_count": s.CourseCount, "total_credits": s.TotalCredit,
			"avg_score": s.AvgScore, "avg_gpa": s.AvgGPA, "pass_rate": s.PassRate(),
		}, nil
	})

	// 3. 竞赛报名（competition_registrations）
	query("competitions", func() (interface{}, error) {
		rows, err := h.profileRepo.ListCompetitions(userID, 10)
		if err != nil {
			return nil, err
		}
		list := make([]gin.H, 0, len(rows))
		for _, row := range rows {
			list = append(list, gin.H{
				"competition_id": row.CompetitionID, "student_name": row.StudentName,
				"team_name": row.TeamName, "advisor_name": row.AdvisorName,
				"status": row.Status, "award_level": row.AwardLevel,
			})
		}
		return list, nil
	})

	// 4. 入党进度（party_progress）
	query("party", func() (interface{}, error) {
		p, err := h.profileRepo.GetPartyProgress(userID)
		if err != nil {
			return nil, err
		}
		return gin.H{"current_stage": p.CurrentStage, "status": p.Status, "apply_date": p.ApplyDate}, nil
	})

	// 5. 社团参与（club_members）
	query("clubs", func() (interface{}, error) {
		rows, err := h.profileRepo.ListClubs(userID, 5)
		if err != nil {
			return nil, err
		}
		list := make([]gin.H, 0, len(rows))
		for _, row := range rows {
			list = append(list, gin.H{"club_id": row.ClubID, "role": row.Role, "join_date": row.JoinDate})
		}
		return list, nil
	})

	// 6. 打卡记录（student_checkins）
	query("checkin", func() (interface{}, error) {
		s, err := h.profileRepo.GetCheckinSummary(userID)
		if err != nil {
			return nil, err
		}
		return gin.H{"total_days": s.TotalDays, "last_date": s.LastDate}, nil
	})

	// 7. 积分（student_points）
	query("points", func() (interface{}, error) {
		total, err := h.profileRepo.GetPointsTotal(userID)
		if err != nil {
			return nil, err
		}
		return gin.H{"total_points": total}, nil
	})

	wg.Wait()
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": result})
}

// decodeJSON 解析 JSON 字符串字段（辅助）
func decodeJSON(raw string) []interface{} {
	var arr []interface{}
	if raw == "" {
		return arr
	}
	if err := json.Unmarshal([]byte(raw), &arr); err != nil {
		return arr
	}
	return arr
}

// UploadAvatar 上传/更新当前用户头像
// POST /api/v1/user/avatar  （multipart/form-data: file）
// 头像 base64 存 users.avatar_base64（SQLite 跨实例持久）
func (h *StudentHandler) UploadAvatar(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未登录"})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "未获取到上传文件"})
		return
	}
	defer file.Close()

	// 限制 3MB
	if header.Size > 3*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "头像文件超过 3MB 限制"})
		return
	}

	ext := strings.ToLower(filepath.Ext(header.Filename))
	allowed := map[string]bool{".png": true, ".jpg": true, ".jpeg": true, ".webp": true, ".gif": true}
	if !allowed[ext] {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "仅支持 png/jpg/webp/gif 图片"})
		return
	}

	bytes, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取文件失败"})
		return
	}

	mime := "image/png"
	switch ext {
	case ".jpg", ".jpeg":
		mime = "image/jpeg"
	case ".gif":
		mime = "image/gif"
	case ".webp":
		mime = "image/webp"
	}

	encoded := base64.StdEncoding.EncodeToString(bytes)
	if err := h.profileRepo.UpdateAvatar(userCtx.UserID, encoded, mime); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "保存头像失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0, "message": "头像已更新",
		"data": gin.H{"user_id": userCtx.UserID, "size": header.Size, "mime": mime},
	})
}

// ServeAvatar GET /api/v1/user/avatar/:user_id — 返回用户头像图片字节（base64 解码）
func (h *StudentHandler) ServeAvatar(c *gin.Context) {
	userID := c.Param("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "缺少 user_id"})
		return
	}

	b64, mime, err := h.profileRepo.GetAvatar(userID)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	c.Header("Content-Type", mime)
	c.Data(http.StatusOK, mime, raw)
}
