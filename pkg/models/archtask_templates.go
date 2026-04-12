// ArchTask Addition - DO NOT MERGE INTO VIKUNJA CORE
// Architectural project templates for different project types.
package models

import (
	"time"

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/log"
	"code.vikunja.io/api/pkg/web"

	"xorm.io/xorm"
)

// ArchTaskTemplate defines a template task for a specific project type and phase.
type ArchTaskTemplate struct {
	Title     string
	Phase     string
	Priority  int64
	DueDays   int // days from project start
	Labels    []string
	Desc      string
}

// architecturalTemplates maps project types to their standard task templates across all phases.
var architecturalTemplates = map[string][]ArchTaskTemplate{
	"residential": {
		// SD Phase
		{Title: "تحليل الموقع وتوثيق الحدود والارتدادات", Phase: "SD", Priority: 5, DueDays: 7, Labels: []string{"موقع", "كود-مصري"}, Desc: "رفع الموقع وتوثيق الحدود الرسمية والارتدادات المطلوبة وفق الكود المصري للمباني السكنية."},
		{Title: "دراسة التوجيه الشمسي وإمكانية التهوية الطبيعية", Phase: "SD", Priority: 4, DueDays: 10, Labels: []string{"استدامة", "SD"}, Desc: "تحليل مسار الشمس والرياح السائدة للوصول لأفضل توجيه للمبنى وتقليل الأحمال الحرارية."},
		{Title: "المسقط الأفقي المبدئي - مقياس 1:200", Phase: "SD", Priority: 5, DueDays: 14, Labels: []string{"رسومات", "SD"}, Desc: "رسم المسقط الأفقي المبدئي موضحاً توزيع الوحدات، المسالك، والخدمات."},
		{Title: "الواجهات الأولية للمشروع - 4 اتجاهات", Phase: "SD", Priority: 4, DueDays: 21, Labels: []string{"رسومات", "SD"}, Desc: "تصميم الواجهات الأولية للجهات الأربع مع اقتراح النظام الإنشائي."},
		// DD Phase
		{Title: "لوحات تفاصيل الواجهة ومواد التشطيب الخارجي", Phase: "DD", Priority: 4, DueDays: 35, Labels: []string{"تشطيبات", "DD"}, Desc: "إعداد كتيب مواد الواجهة مع المواصفات والموردين المقترحين."},
		{Title: "مخطط الغرف الرطبة والصرف الصحي - DD", Phase: "DD", Priority: 5, DueDays: 40, Labels: []string{"صرف-صحي", "DD"}, Desc: "تفاصيل الحمامات والمطابخ مع مسارات الأنابيب والفتحات."},
		{Title: "تصميم نظام التكييف والتهوية الميكانيكي", Phase: "DD", Priority: 3, DueDays: 42, Labels: []string{"ميكانيكي", "DD"}, Desc: "تحديد نظام التكييف المناسب وحساب الأحمال الحرارية لكل وحدة."},
		// CD Phase
		{Title: "لوحات الإنشاء الكاملة - هيكل وأساسات", Phase: "CD", Priority: 5, DueDays: 60, Labels: []string{"إنشاء", "CD"}, Desc: "لوحات شاملة لنظام الأساسات والهيكل الإنشائي معتمدة من المهندس الإنشائي."},
		{Title: "جدول الأبواب والنوافذ مع المواصفات", Phase: "CD", Priority: 4, DueDays: 65, Labels: []string{"نجارة", "CD"}, Desc: "جدول تفصيلي بجميع الأبواب والنوافذ مع الأبعاد والمواد والأقفال."},
		{Title: "مواصفات المقاول وبنود العقد", Phase: "CD", Priority: 5, DueDays: 70, Labels: []string{"عقود", "CD"}, Desc: "إعداد كراسة المواصفات الفنية ووثائق العطاء للمقاول العام."},
		// CA Phase
		{Title: "اجتماع الإحالة وتسليم الموقع للمقاول", Phase: "CA", Priority: 5, DueDays: 80, Labels: []string{"إشراف", "CA"}, Desc: "اجتماع القدم الأول مع المقاول وتوثيق تسليم الموقع رسمياً."},
		{Title: "تقارير الإشراف الأسبوعية", Phase: "CA", Priority: 4, DueDays: 90, Labels: []string{"إشراف", "CA"}, Desc: "متابعة التنفيذ وإعداد تقارير الإشراف الأسبوعية بالصور والملاحظات."},
	},
	"commercial": {
		{Title: "دراسة الجدوى المعمارية والاستيعابية", Phase: "SD", Priority: 5, DueDays: 7, Labels: []string{"جدوى", "SD"}, Desc: "تحليل الطاقة الاستيعابية القصوى للمبنى التجاري وفق الكود المصري واشتراطات الحريق."},
		{Title: "مخطط توزيع الفراغات التجارية - SD", Phase: "SD", Priority: 5, DueDays: 14, Labels: []string{"رسومات", "SD"}, Desc: "تصميم مبدئي لتوزيع المحلات التجارية، الممرات، ومداخل الطوارئ."},
		{Title: "دراسة نظام الواجهات الزجاجية الستائرية", Phase: "DD", Priority: 4, DueDays: 35, Labels: []string{"واجهات", "DD"}, Desc: "اختيار نظام الواجهات المناسب مع الدراسة الحرارية واختبارات النفاذية."},
		{Title: "تصميم نظام HVAC المركزي", Phase: "DD", Priority: 5, DueDays: 42, Labels: []string{"ميكانيكي", "DD"}, Desc: "تصميم نظام التكييف المركزي وحساب الأحمال الحرارية الكاملة."},
		{Title: "لوحات مسارات الإخلاء ومكافحة الحريق", Phase: "CD", Priority: 5, DueDays: 60, Labels: []string{"حريق", "CD"}, Desc: "لوحات تفصيلية لنظام مكافحة الحريق والإخلاء معتمدة من الدفاع المدني."},
		{Title: "وثائق عطاء المقاول العام", Phase: "CD", Priority: 5, DueDays: 70, Labels: []string{"عقود", "CD"}, Desc: "إعداد وثائق العطاء الكاملة للمقاول العام والمقاولين المتخصصين."},
		{Title: "الاستلام الابتدائي ورصد العيوب", Phase: "CA", Priority: 5, DueDays: 120, Labels: []string{"تسليم", "CA"}, Desc: "فحص المشروع وإعداد قائمة العيوب قبل الاستلام الابتدائي الرسمي."},
	},
	"institutional": {
		{Title: "مراجعة اشتراطات الجهة المعنية (وزارة/هيئة)", Phase: "SD", Priority: 5, DueDays: 5, Labels: []string{"اشتراطات", "SD"}, Desc: "توثيق جميع اشتراطات الجهة الحكومية المختصة ومتطلبات البرنامج الوظيفي."},
		{Title: "تصميم سبل الوصول للمعاقين - ADA/ECP", Phase: "SD", Priority: 5, DueDays: 14, Labels: []string{"إتاحة", "SD"}, Desc: "تصميم منظومة الوصول الشامل للأشخاص ذوي الإعاقة وفق الكود المصري والمعايير الدولية."},
		{Title: "تصميم أنظمة الصوت والإضاءة الخاصة", Phase: "DD", Priority: 4, DueDays: 40, Labels: []string{"أنظمة", "DD"}, Desc: "تصميم أنظمة الصوت والإضاءة الخاصة بالفراغات الوظيفية (قاعات، مصليات، إلخ)."},
		{Title: "مواصفات المناقصة الحكومية الرسمية", Phase: "CD", Priority: 5, DueDays: 65, Labels: []string{"مناقصة", "CD"}, Desc: "إعداد وثائق المناقصة وفق قواعد المناقصات والمزايدات الحكومية المصرية."},
		{Title: "محضر لجنة الاستلام الحكومية", Phase: "CA", Priority: 5, DueDays: 110, Labels: []string{"استلام", "CA"}, Desc: "التحضير للجنة الاستلام الحكومية وتوثيق المطابقة مع الرسومات المعتمدة."},
	},
	"default": {
		{Title: "تحليل الموقع والمتطلبات", Phase: "SD", Priority: 5, DueDays: 7, Labels: []string{"تحليل", "SD"}, Desc: "تحليل شامل للموقع والمتطلبات الوظيفية والقانونية للمشروع."},
		{Title: "المسقط الأفقي المبدئي", Phase: "SD", Priority: 5, DueDays: 14, Labels: []string{"رسومات", "SD"}, Desc: "المسقط الأفقي المبدئي موضحاً توزيع الفراغات الرئيسية."},
		{Title: "تفاصيل الواجهات والمواد", Phase: "DD", Priority: 4, DueDays: 35, Labels: []string{"تشطيبات", "DD"}, Desc: "تفاصيل الواجهات واختيار مواد التشطيب الخارجي والداخلي."},
		{Title: "لوحات الإنشاء والتفاصيل الكاملة", Phase: "CD", Priority: 5, DueDays: 60, Labels: []string{"إنشاء", "CD"}, Desc: "المجموعة الكاملة من اللوحات التنفيذية والمواصفات."},
		{Title: "الإشراف على التنفيذ والتسليم", Phase: "CA", Priority: 5, DueDays: 100, Labels: []string{"إشراف", "CA"}, Desc: "متابعة التنفيذ وإجراءات الاستلام الرسمية."},
	},
}

// ApplyArchitecturalTemplate creates standard tasks for a project based on its type.
// It uses a new DB session internally and is safe to call as a standalone operation.
func ApplyArchitecturalTemplate(projectID int64, projectType string, auth web.Auth) (int, error) {
	s := db.NewSession()
	defer s.Close()

	if err := s.Begin(); err != nil {
		return 0, err
	}

	templates, ok := architecturalTemplates[projectType]
	if !ok {
		templates = architecturalTemplates["default"]
		log.Infof("[ArchTask] Unknown project type '%s', using default templates", projectType)
	}

	created := 0
	now := time.Now()

	for _, tmpl := range templates {
		dueDate := now.AddDate(0, 0, tmpl.DueDays)
		task := &Task{
			Title:       tmpl.Title,
			Description: tmpl.Desc,
			ProjectID:   projectID,
			Priority:    tmpl.Priority,
			DueDate:     dueDate,
			ArchPhase:   tmpl.Phase,
			AIGenerated: false,
		}

		if err := createTask(s, task, auth, false, false); err != nil {
			log.Errorf("[ArchTask] Failed to create template task '%s': %v", tmpl.Title, err)
			continue
		}
		created++
	}

	if err := s.Commit(); err != nil {
		_ = s.Rollback()
		return 0, err
	}

	log.Infof("[ArchTask] Applied %d template tasks for project %d (type: %s)", created, projectID, projectType)
	return created, nil
}

// GetSupportedProjectTypes returns the list of supported architectural project types.
func GetSupportedProjectTypes() []map[string]string {
	return []map[string]string{
		{"value": "residential", "label_ar": "سكني", "label_en": "Residential"},
		{"value": "commercial", "label_ar": "تجاري", "label_en": "Commercial"},
		{"value": "institutional", "label_ar": "مؤسسي", "label_en": "Institutional"},
		{"value": "industrial", "label_ar": "صناعي", "label_en": "Industrial"},
		{"value": "default", "label_ar": "عام", "label_en": "General"},
	}
}

// GetProjectPhaseSummary returns task counts per arch phase for a given project.
func GetProjectPhaseSummary(s *xorm.Session, projectID int64) (map[string]map[string]int64, error) {
	type phaseStat struct {
		ArchPhase string `xorm:"arch_phase"`
		Done      bool   `xorm:"done"`
		Count     int64  `xorm:"count"`
	}

	var stats []phaseStat
	err := s.Table("tasks").
		Select("arch_phase, done, COUNT(*) as count").
		Where("project_id = ?", projectID).
		And("arch_phase != ''").
		GroupBy("arch_phase, done").
		Find(&stats)
	if err != nil {
		return nil, err
	}

	result := map[string]map[string]int64{
		"SD": {"total": 0, "done": 0},
		"DD": {"total": 0, "done": 0},
		"CD": {"total": 0, "done": 0},
		"CA": {"total": 0, "done": 0},
	}

	for _, stat := range stats {
		if _, exists := result[stat.ArchPhase]; !exists {
			result[stat.ArchPhase] = map[string]int64{"total": 0, "done": 0}
		}
		result[stat.ArchPhase]["total"] += stat.Count
		if stat.Done {
			result[stat.ArchPhase]["done"] += stat.Count
		}
	}

	return result, nil
}
