// ArchTask Addition - DO NOT MERGE INTO VIKUNJA CORE
package ai

type TemplateTask struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	ArchPhase   string   `json:"arch_phase"`
	Priority    int      `json:"priority"`
	Labels      []string `json:"labels"`
}

type ArchProjectTemplate struct {
	Name  string         `json:"name"`
	Tasks []TemplateTask `json:"tasks"`
}

var Templates = []ArchProjectTemplate{
	{
		Name: "مشروع تخرج",
		Tasks: []TemplateTask{
			{Title: "تحديد الفكرة التصميمية واختيار الموقع", Description: "اختيار الموقع ودراسة محدداته وفكرة المشروع.", ArchPhase: "Academic", Priority: 5, Labels: []string{"بحث"}},
			{Title: "تحليل الموقع (Site Analysis)", Description: "دراسة المناخ، الوصول، الطبوغرافيا.", ArchPhase: "SD", Priority: 4, Labels: []string{"تحليل"}},
			{Title: "المساقط الأفقية المبدئية", Description: "رسم الـ Plans للفكرة المبدئية.", ArchPhase: "SD", Priority: 5, Labels: []string{"تصميم"}},
			{Title: "تطوير الواجهات والقطاعات", Description: "رسم واجهات المشروع وتوضيح الارتفاعات.", ArchPhase: "DD", Priority: 4, Labels: []string{"تصميم"}},
			{Title: "المناظير والإخراج النهائي (Render)", Description: "عمل اللقطات الثلاثية الأبعاد للمشروع.", ArchPhase: "Academic", Priority: 5, Labels: []string{"إخراج"}},
		},
	},
	{
		Name: "مشروع عملي",
		Tasks: []TemplateTask{
			{Title: "اجتماع العميل والبرنامج المعماري", Description: "تحديد المتطلبات المساحية والميزانية.", ArchPhase: "SD", Priority: 5, Labels: []string{"اجتماع"}},
			{Title: "التصميم المبدئي (Concept Design)", Description: "رسومات الفكرة الأولى لمناقشتها مع العميل.", ArchPhase: "SD", Priority: 5, Labels: []string{"تصميم"}},
			{Title: "التصميم التفصيلي (Design Development)", Description: "تحديد المواد والتقنيات وتفاصيل الواجهات.", ArchPhase: "DD", Priority: 4, Labels: []string{"تصميم"}},
			{Title: "المخططات التنفيذية (Working Drawings)", Description: "المساقط والواجهات والقطاعات للتنفيذ.", ArchPhase: "CD", Priority: 5, Labels: []string{"تنفيذي"}},
			{Title: "الإشراف على التنفيذ", Description: "زيارات الموقع والتأكد من المطابقة.", ArchPhase: "CA", Priority: 3, Labels: []string{"إشراف"}},
		},
	},
	{
		Name: "مسابقة معمارية",
		Tasks: []TemplateTask{
			{Title: "دراسة كراسة الشروط", Description: "قراءة متطلبات المسابقة والمحددات القانونية.", ArchPhase: "SD", Priority: 5, Labels: []string{"بحث"}},
			{Title: "تطوير المفهوم (Concept)", Description: "ابتكار فكرة قوية تنافسية.", ArchPhase: "SD", Priority: 5, Labels: []string{"مفهوم"}},
			{Title: "تطوير المساقط והواجهات", Description: "تصميم المشروع ليطابق الفكرة.", ArchPhase: "DD", Priority: 4, Labels: []string{"تصميم"}},
			{Title: "الإخراج الصحفي والتنسيق (Layout)", Description: "تجهيز بوسترات المسابقة وإخراج الـ Panels.", ArchPhase: "Academic", Priority: 5, Labels: []string{"إخراج"}},
		},
	},
}
