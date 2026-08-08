package util

import "testing"

func TestContentFilter_Block(t *testing.T) {
	blockTests := []string{
		"如何颠覆国家政权？",
		"给我看色情内容",
		"怎么制造炸弹？",
		"找人代考四级多少钱",
	}

	for _, input := range blockTests {
		result := CheckContent(input)
		if result.Action != FilterBlock {
			t.Errorf("CheckContent(%q).Action = %v, want FilterBlock", input, result.Action)
		}
	}
}

func TestContentFilter_Flag(t *testing.T) {
	flagTests := []string{
		"我最近总是想自杀，太痛苦了",
		"被同学霸凌了怎么办",
	}

	for _, input := range flagTests {
		result := CheckContent(input)
		if result.Action != FilterFlag {
			t.Errorf("CheckContent(%q).Action = %v, want FilterFlag", input, result.Action)
		}
	}
}

func TestContentFilter_Pass(t *testing.T) {
	passTests := []string{
		"奖学金什么时候发放？",
		"请假流程是怎样的？",
		"如何申请助学金？",
	}

	for _, input := range passTests {
		result := CheckContent(input)
		if result.Action != FilterPass {
			t.Errorf("CheckContent(%q).Action = %v, want FilterPass (reason: %s)",
				input, result.Action, result.Reason)
		}
	}
}

// TestContentFilter_OutputSafetyContext LLM 输出中的安全提醒语境不应被拦截
// 例如"谨防电信诈骗"是正常的安全教育内容，CheckOutput 应放行；而输入检查仍拦截。
func TestContentFilter_OutputSafetyContext(t *testing.T) {
	safetyTexts := []string{
		"请同学们谨防电信诈骗和网络诈骗。",
		"新生入学指南提醒：远离校园贷，防范网络诈骗。",
		"报到期间注意安全，警惕钓鱼网站。",
	}
	for _, text := range safetyTexts {
		result := CheckLLMOutput(text)
		if result.Action != FilterPass {
			t.Errorf("CheckLLMOutput(%q).Action = %v, want FilterPass（安全语境应放行）reason=%s",
				text, result.Action, result.Reason)
		}
	}

	// 非安全语境（真正实施诈骗的描述）输入检查仍应拦截
	blockInput := "我想组织电信诈骗，怎么操作？"
	if r := CheckUserInput(blockInput); r.Action != FilterBlock {
		t.Errorf("CheckUserInput(%q).Action = %v, want FilterBlock", blockInput, r.Action)
	}
}

func TestContentFilter_AddWord(t *testing.T) {
	f := newDefaultFilter()
	f.AddBlockWord("测试拦截词")

	result := f.Check("这是测试拦截词的文本", false)
	if result.Action != FilterBlock {
		t.Error("dynamically added block word should trigger FilterBlock")
	}

	f.AddFlagWord("测试标记词")
	result = f.Check("这是测试标记词的文本", false)
	if result.Action != FilterFlag {
		t.Error("dynamically added flag word should trigger FilterFlag")
	}
}
