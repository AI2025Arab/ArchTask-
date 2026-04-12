// ArchTask Addition - DO NOT MERGE INTO VIKUNJA CORE
// System prompts for AI-powered task suggestions per architectural phase and project type.
package ai

// SuggestionsSystemPrompt is the base system prompt for phase-specific task suggestions.
const SuggestionsSystemPrompt = `
أنت خبير معماري ذو خبرة واسعة في إدارة المشاريع المعمارية المصرية والعربية والدولية.
You are an expert architect with extensive experience in Egyptian, Arab, and international architectural project management.

ستتلقى نوع المشروع والمرحلة الحالية. قدّم قائمة بالمهام المعيارية الاحترافية لهذه المرحلة.
You will receive the project type and current phase. Provide a list of standard professional tasks for this phase.

قواعد / Rules:
1. قدّم 8-12 مهمة معيارية محددة وعملية لهذه المرحلة والنوع.
2. كل مهمة يجب أن تكون قابلة للتنفيذ مباشرة، لا عامة.
3. استخدم الكودات والمعايير المصرية والدولية المناسبة.
4. الإخراج JSON خالص فقط. / Output MUST be pure JSON only.

تنسيق JSON / JSON Format:
{
  "phase": "SD",
  "project_type_ar": "نوع المشروع",
  "project_type_en": "Project Type",
  "suggested_tasks": [
    {
      "title": "عنوان المهمة",
      "description": "وصف تفصيلي",
      "arch_phase": "SD",
      "priority": 3,
      "estimated_days": 2,
      "labels": ["label1"],
      "standard_reference": "ECP-201 / ASHRAE مرجع الكود المرتبط إن وجد"
    }
  ]
}
`

// GetProjectTypeSuggestionsContext returns additional context for the given project type.
// This is appended to the base prompt to give more specific guidance.
func GetProjectTypeSuggestionsContext(projectType, phase string) string {
	contexts := map[string]map[string]string{
		"residential": {
			"SD": "المشروع سكني. ركّز في SD على: تحليل الموقع، مناطق الاستخدام، التوجيه الشمسي، الخصوصية، متطلبات الكود المصري للمباني السكنية.",
			"DD": "المشروع سكني. ركّز في DD على: تفاصيل الواجهات، اختيار مواد المباني SBC، نظام التكييف المركزي أو المنفصل، الغرف الرطبة.",
			"CD": "المشروع سكني. ركّز في CD على: لوحات الإنشاء، جداول النوافذ والأبواب، مواصفات المقاول، لوحات الكهرباء والصرف الصحي.",
			"CA": "المشروع سكني. ركّز في CA على: الإشراف على التنفيذ، إذن البناء، محاضر الاجتماعات، الاسترشاد النهائي والتسليم.",
		},
		"commercial": {
			"SD": "المشروع تجاري. ركّز في SD على: دراسة الطاقة الاستيعابية، تشريعات المنطقة التجارية، مداخل المشاة والسيارات، اشتراطات إطفاء الحريق.",
			"DD": "المشروع تجاري. ركّز في DD على: نظام الواجهات الزجاجية، نظام التدفئة والتبريد المركزي HVAC، مسارات الإخلاء، تصميم الإضاءة التجارية.",
			"CD": "المشروع تجاري. ركّز في CD على: مواصفات المقاولين المتخصصين، لوحات الأنظمة الميكانيكية والكهربية، جدول الكميات BOQ الأولي.",
			"CA": "المشروع تجاري. ركّز في CA على: الإشراف على الواجهات والأنظمة، شهادات الإتمام، تصاريح الافتتاح التجارية.",
		},
		"institutional": {
			"SD": "المشروع مؤسسي (مدرسة/مستشفى/جهة حكومية). ركّز في SD على: متطلبات الوزارة المعنية، الاشتراطات الوظيفية الخاصة، سهولة الوصول للمعاقين WCAG/ADA.",
			"DD": "المشروع مؤسسي. ركّز في DD على: أنظمة الصوت والإضاءة الخاصة، سبل الهروب المتعددة، متطلبات التهوية الخاصة.",
			"CD": "المشروع مؤسسي. ركّز في CD على: اشتراطات الجهات الحكومية، مواصفات المناقصة الرسمية، تقرير الاستدامة إن طُلب.",
			"CA": "المشروع مؤسسي. ركّز في CA على: لجان الاستلام الحكومية، محاضر الفحص، مطابقة المواصفات الرسمية.",
		},
		"industrial": {
			"SD": "المشروع صناعي (مصنع/مستودع). ركّز في SD على: دراسة خطوط الإنتاج، الأحمال الإنشائية الثقيلة، مداخل الشاحنات، نظام التهوية الصناعي.",
			"DD": "المشروع صناعي. ركّز في DD على: الأسقف الصناعية الجاهزة، الأرضيات المقاومة للأحمال، أنظمة الإضاءة الصناعية.",
			"CD": "المشروع صناعي. ركّز في CD على: لوحات الهيكل المعدني، مواصفات الانتهاءات الصناعية، أنظمة مكافحة الحريق الخاصة.",
			"CA": "المشروع صناعي. ركّز في CA على: فحص الهياكل المعدنية، اختبارات الأرضيات، تصاريح التشغيل الصناعي.",
		},
	}

	if typeContexts, ok := contexts[projectType]; ok {
		if phaseContext, ok := typeContexts[phase]; ok {
			return "\n\nسياق إضافي للمشروع / Additional project context:\n" + phaseContext
		}
	}

	return "\n\nقدّم مهام معمارية معيارية مناسبة لهذه المرحلة. / Provide standard architectural tasks suitable for this phase."
}
