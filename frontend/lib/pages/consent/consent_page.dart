import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import '../../utils/storage.dart';

/// 首次启动隐私同意页面
/// 在登录前展示，用户必须同意隐私政策和用户协议后方可进入应用
class ConsentPage extends StatefulWidget {
  const ConsentPage({super.key});

  @override
  State<ConsentPage> createState() => _ConsentPageState();
}

class _ConsentPageState extends State<ConsentPage> {
  bool _privacyAccepted = false;
  bool _termsAccepted = false;
  bool _submitting = false;

  bool get _allAccepted => _privacyAccepted && _termsAccepted;

  Future<void> _onAgree() async {
    if (!_allAccepted || _submitting) return;
    setState(() => _submitting = true);

    await Storage.setConsented(true);
    await Storage.setFirstLaunchDone();

    if (!mounted) return;
    // 跳转到登录页
    context.go('/login');
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final isDesktop = MediaQuery.of(context).size.width > 600;

    return Scaffold(
      body: SafeArea(
        child: Center(
          child: SingleChildScrollView(
            padding: const EdgeInsets.symmetric(horizontal: 32, vertical: 48),
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 560),
              child: Column(
                mainAxisAlignment: MainAxisAlignment.center,
                children: [
                  // Logo 区域
                  Icon(Icons.school, size: 64, color: theme.colorScheme.primary),
                  const SizedBox(height: 12),
                  Text(
                    '欢迎使用蔚小芯',
                    style: theme.textTheme.headlineSmall?.copyWith(
                      fontWeight: FontWeight.bold,
                      color: theme.colorScheme.primary,
                    ),
                  ),
                  const SizedBox(height: 8),
                  Text(
                    '信息学院智慧学工 AI 助手',
                    style: theme.textTheme.bodyLarge?.copyWith(
                      color: theme.colorScheme.onSurfaceVariant,
                    ),
                  ),
                  const SizedBox(height: 32),

                  // 隐私政策说明卡片
                  Card(
                    elevation: 0,
                    shape: RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(16),
                      side: BorderSide(
                        color: theme.colorScheme.outlineVariant.withOpacity( 0.5),
                      ),
                    ),
                    child: Padding(
                      padding: const EdgeInsets.all(24),
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Row(
                            children: [
                              Icon(Icons.privacy_tip_outlined,
                                  color: theme.colorScheme.primary, size: 24),
                              const SizedBox(width: 8),
                              Text(
                                '在使用前，请阅读并同意以下协议',
                                style: theme.textTheme.titleSmall?.copyWith(
                                  fontWeight: FontWeight.w600,
                                ),
                              ),
                            ],
                          ),
                          const SizedBox(height: 20),

                          // 隐私政策勾选
                          _buildAgreementItem(
                            theme: theme,
                            title: '《蔚小芯隐私政策》',
                            subtitle: '了解我们如何收集、使用和保护你的个人信息',
                            value: _privacyAccepted,
                            onChanged: (v) => setState(() => _privacyAccepted = v ?? false),
                            content: _privacyPolicyContent,
                          ),
                          const Divider(height: 24),

                          // 用户协议勾选
                          _buildAgreementItem(
                            theme: theme,
                            title: '《蔚小芯用户协议》',
                            subtitle: '了解服务规则、免责声明与使用规范',
                            value: _termsAccepted,
                            onChanged: (v) => setState(() => _termsAccepted = v ?? false),
                            content: _termsOfServiceContent,
                          ),
                        ],
                      ),
                    ),
                  ),

                  const SizedBox(height: 16),

                  // 数据安全说明
                  Card(
                    elevation: 0,
                    color: theme.colorScheme.primaryContainer.withOpacity( 0.3),
                    shape: RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(12),
                    ),
                    child: Padding(
                      padding: const EdgeInsets.all(16),
                      child: Row(
                        children: [
                          Icon(Icons.shield_outlined,
                              color: theme.colorScheme.primary, size: 20),
                          const SizedBox(width: 12),
                          Expanded(
                            child: Text(
                              '你的数据安全是我们的首要任务。所有敏感信息（学号、手机号、身份证号等）在传输和存储过程中均经过加密和脱敏处理。',
                              style: theme.textTheme.bodySmall?.copyWith(
                                color: theme.colorScheme.onSurfaceVariant,
                              ),
                            ),
                          ),
                        ],
                      ),
                    ),
                  ),

                  const SizedBox(height: 32),

                  // 同意按钮
                  SizedBox(
                    width: isDesktop ? 320 : double.infinity,
                    height: 48,
                    child: FilledButton(
                      onPressed: _allAccepted && !_submitting ? _onAgree : null,
                      child: _submitting
                          ? const SizedBox(
                              width: 24,
                              height: 24,
                              child: CircularProgressIndicator(
                                  strokeWidth: 2, color: Colors.white),
                            )
                          : const Text('同意并继续', style: TextStyle(fontSize: 16)),
                    ),
                  ),

                  const SizedBox(height: 12),
                  Text(
                    '同意后即可使用全部功能。你可以随时在"个人中心"查看协议内容。',
                    style: theme.textTheme.bodySmall?.copyWith(
                      color: theme.colorScheme.onSurfaceVariant,
                    ),
                    textAlign: TextAlign.center,
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }

  Widget _buildAgreementItem({
    required ThemeData theme,
    required String title,
    required String subtitle,
    required bool value,
    required ValueChanged<bool?> onChanged,
    required String content,
  }) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        CheckboxListTile(
          value: value,
          onChanged: onChanged,
          title: Text(title, style: const TextStyle(fontWeight: FontWeight.w500)),
          subtitle: Text(subtitle),
          controlAffinity: ListTileControlAffinity.leading,
          contentPadding: EdgeInsets.zero,
          dense: true,
        ),
        const SizedBox(height: 4),
        Padding(
          padding: const EdgeInsets.only(left: 12),
          child: GestureDetector(
            onTap: () => _showContentDialog(title, content),
            child: Text(
              '点击查看详情 →',
              style: theme.textTheme.labelMedium?.copyWith(
                color: theme.colorScheme.primary,
                decoration: TextDecoration.underline,
              ),
            ),
          ),
        ),
      ],
    );
  }

  void _showContentDialog(String title, String content) {
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(title),
        content: SizedBox(
          width: double.maxFinite,
          child: SingleChildScrollView(
            child: Text(content, style: const TextStyle(fontSize: 13, height: 1.6)),
          ),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(),
            child: const Text('关闭'),
          ),
        ],
      ),
    );
  }

  static const _privacyPolicyContent = '''
蔚小芯隐私政策

更新日期：2026年5月17日
生效日期：2026年5月17日

一、信息收集

1.1 我们收集的信息类型：
• 身份信息：学号/工号、姓名、学院、专业、班级
• 联系方式：手机号码（用于重要通知）
• 学习数据：课程表、成绩、选课记录、学习进度
• 使用数据：问答记录、功能使用频率、会话历史
• 情感数据：心情打卡记录、心理评估结果（可选，仅用于关怀服务）

1.2 收集方式：
• 用户主动提供：登录认证、表单填写、语音输入
• 系统自动采集：功能使用日志、设备信息（用于兼容性优化）
• 学校系统同步：经授权从学工系统、教务系统获取

二、信息使用

2.1 我们使用收集的信息用于：
• 提供个性化 AI 学工服务（课程推荐、学业预警、心理关怀）
• 辅导员工作支持（学生状态监测、谈心谈话辅助）
• 教学质量分析与改进
• 校园服务优化（食堂推荐、图书馆座位查询）

2.2 自动化决策说明：
• AI 今日速览、学业预警、情感预警等功能使用自动化算法
• 所有 AI 建议仅供参考，不作为唯一决策依据
• 重要决策（如处分、退学建议）必须由人工审核确认

三、信息保护

3.1 安全措施：
• 传输加密：所有网络通信使用 HTTPS/TLS 加密
• 存储加密：敏感数据在数据库中加密存储
• PII 脱敏：学号、手机号、身份证号在与 AI 模型交互前自动脱敏
• 访问控制：基于角色的六级权限控制（RBAC）
• 审计日志：所有数据访问操作记录可追溯

3.2 数据保留：
• 账号有效期内保留数据
• 毕业后可选择导出或删除个人数据
• 匿名化数据可能用于长期教育研究

四、信息共享

4.1 我们不会：
• 向第三方出售你的个人信息
• 未经授权向校外机构提供你的数据
• 将你的数据用于商业广告推送

4.2 信息共享场景：
• 辅导员查看管辖学生的学情数据和情感状态
• 教师查看授课班级的学情热力图（匿名化）
• 学校管理员查看全校统计数据（匿名化）
• 法律要求披露时

五、你的权利

• 知情权：了解我们收集了哪些你的信息
• 访问权：查看和导出你的个人数据
• 更正权：发现信息错误时可以申请更正
• 删除权：毕业离校后可申请删除个人数据
• 撤回同意权：可随时撤回同意，但可能影响功能使用

六、未成年人保护

• 对于未满18周岁的用户，我们建议在监护人的指导下使用本应用
• 如发现未经监护人同意收集了未成年人信息，我们将及时删除

七、政策更新

• 隐私政策更新时会通过应用内弹窗通知
• 重大变更需要重新获得你的同意
• 继续使用即表示你同意更新后的政策

八、联系方式

• 信息中心：图书馆一楼信息中心办公室
• 联系电话：0550-XXXXXXX
• 电子邮箱：wxx@chzu.edu.cn''';

  static const _termsOfServiceContent = '''
蔚小芯用户协议

更新日期：2026年5月17日

一、服务说明

1.1 蔚小芯是滁州学院信息学院提供的智慧校园 AI 学工助手，旨在为师生提供便捷的校园信息查询、学习辅导和生活服务。

1.2 服务功能包括但不限于：
• 智能问答（政策咨询、流程引导、学习辅导）
• 个性化推荐（课程推荐、活动推荐、学习资源推荐）
• 学情分析（学业预警、情感关怀、成长规划）
• 校园生活服务（食堂、图书馆、校车、报修等）

二、用户义务

2.1 用户在使用过程中应当：
• 提供真实、准确的个人信息
• 妥善保管账号和密码，不得转借他人使用
• 不得利用本服务从事违法违规活动
• 不得恶意攻击、干扰系统正常运行
• 不得通过本服务传播违法和不良信息

2.2 用户不得：
• 使用自动化工具批量发送请求（爬虫、刷接口）
• 尝试绕过权限控制访问他人数据
• 上传包含恶意代码的文件
• 利用 AI 服务生成违法违规内容

三、免责声明

3.1 AI 回答准确性：
• 蔚小芯提供的 AI 回答基于知识库和语言模型生成
• 对于政策类信息，我们力求准确，但最终解释权归学校相关部门
• 用户在做出重要决策前应核实相关信息

3.2 服务可用性：
• 我们努力保证服务的连续性和稳定性
• 因系统维护、网络故障、不可抗力等原因可能导致服务中断
• 我们不对因服务中断造成的间接损失承担责任

3.3 第三方服务：
• 本应用可能包含第三方大模型 API（智谱清言、DeepSeek、讯飞星火）
• 第三方服务的可用性和准确性由其提供商负责
• 我们已对发送给第三方的内容进行了 PII 脱敏处理

四、知识产权

4.1 蔚小芯的软件著作权、商标等知识产权归开发方所有。
4.2 用户在应用中产生的内容（问答、反馈等）默认授权平台用于改进服务质量。
4.3 知识库中收录的学校政策文件的版权归滁州学院所有。

五、违规处理

5.1 如发现用户违反本协议，我们有权：
• 警告并记录违规行为
• 限制或暂停违规功能的使用
• 上报学校相关部门处理
• 终止严重违规者的账号

六、协议变更

• 协议更新时会通过应用内通知
• 重大变更需要重新获得你的同意
• 继续使用即表示你同意更新后的协议

七、适用法律

• 本协议适用中华人民共和国法律
• 因本协议产生的争议，双方应友好协商解决
• 协商不成的，提交有管辖权的人民法院处理''';
}
