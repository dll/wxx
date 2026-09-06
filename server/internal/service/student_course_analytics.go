package service

import "context"

// GenerateCourseAnalytics is the stable course analytics entry point. The legacy
// implementation remains available internally while this boundary is migrated.
func (s *StudentService) GenerateCourseAnalytics(ctx context.Context, userID int64) (*CourseAnalyticsResult, error) {
	if s.twinRepo == nil || s.userRepo == nil {
		return nil, nil
	}
	user, err := s.userRepo.GetByID(userID)
	if err != nil || user == nil {
		return nil, nil
	}
	grades, err := s.twinRepo.ListCourseGrades(userID)
	if err != nil || len(grades) == 0 {
		return nil, err
	}
	basis, _ := s.twinRepo.GetClassBasis(user.ClassName)
	res := &CourseAnalyticsResult{UserDisplayName: user.DisplayName, ClassName: user.ClassName, Courses: make([]*CourseAnalyticsItem, 0, len(grades)), WeakCourses: []string{}, DataSource: "real"}
	if basis != nil {
		res.ClassAvgGPA, res.ClassSize = basis.ClassAvgGPA, basis.ClassSize
	}
	var gpaSum float64
	for _, g := range grades {
		item := &CourseAnalyticsItem{CourseName: g.CourseName, Semester: g.Semester, Score: g.Score, GPA: g.GPA, GradeLevel: g.GradeLevel, Credits: g.Credits, Passed: g.Passed}
		res.Courses = append(res.Courses, item)
		gpaSum += g.GPA
		if g.Passed {
			res.CreditsEarned += g.Credits
		}
		if !g.Passed || g.Score < 70 {
			res.WeakCourses = append(res.WeakCourses, g.CourseName)
		}
	}
	res.OverallGPA = gpaSum / float64(len(grades))
	res.Advice = s.buildCourseAdvice(ctx, res)
	return res, nil
}
