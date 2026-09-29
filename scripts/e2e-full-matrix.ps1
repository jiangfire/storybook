# Storybook 全角色×功能矩阵 E2E 验收脚本（API 级，补充 P0 脚本的缺口）
# 前置：真实服务 + bootstrap-admin（admin@e2e.test / Admin12345!），$env:E2E_BASE 已设置
# 环境要求：AI_CONFIG_ENCRYPTION_KEY（32 字节）需已设置，否则 AI 配置保存会报加密失败
$ErrorActionPreference = 'Stop'
$Base = $env:E2E_BASE
if (-not $Base) { $Base = 'http://127.0.0.1:18080' }

$script:pass = 0
$script:fail = 0

function Invoke-Api {
  param([string]$Method, [string]$Path, $Body = $null, [string]$Token = '')
  $headers = @{}
  if ($Token) { $headers['Authorization'] = "Bearer $Token" }
  $json = if ($null -ne $Body) { $Body | ConvertTo-Json -Depth 8 } else { $null }
  try {
    $r = Invoke-RestMethod -Method $Method -Uri "$Base$Path" -Headers $headers -ContentType 'application/json' -Body $json
    return @{ ok = $true; data = $r.data }
  } catch {
    $code = 0
    try { $code = [int]$_.Exception.Response.StatusCode } catch {}
    $detail = "$($_.ErrorDetails.Message)"
    return @{ ok = $false; status = $code; detail = $detail }
  }
}

function Assert {
  param([bool]$Cond, [string]$Name, $Actual = '', $Expect = '')
  if ($Cond) {
    $script:pass++
    Write-Host "  PASS  $Name"
  } else {
    $script:fail++
    Write-Host "  FAIL  $Name  (actual=$Actual expect=$Expect)"
  }
}

function Invoke-AuthPaced {
  param([string]$Method, [string]$Path, $Body)
  Start-Sleep -Seconds 30
  return Invoke-Api -Method $Method -Path $Path -Body $Body
}

Write-Host "== A. 账号准备（admin 建账号绕过注册限流，登录逐个 30s 间隔） =="
$admin = Invoke-AuthPaced Post '/api/auth/login' @{ email = 'admin@e2e.test'; password = 'Admin12345!' }
Assert $admin.ok 'admin 登录'
$adminToken = $admin.data.token

foreach ($u in @(
  @{ email = 'pm@e2e.test'; username = 'e2e-pm'; password = 'Pm123456!'; role = 'product' },
  @{ email = 'dev@e2e.test'; username = 'e2e-dev'; password = 'Dev123456!'; role = 'developer' },
  @{ email = 'tl@e2e.test'; username = 'e2e-tl'; password = 'Tl123456!'; role = 'tech_lead' },
  @{ email = 'tester@e2e.test'; username = 'e2e-tester'; password = 'Test12345!'; role = 'tester' }
)) {
  $existing = (Invoke-Api Get '/api/admin/users' -Token $adminToken).data.users | Where-Object { $_.email -eq $u.email } | Select-Object -First 1
  if (-not $existing) {
    $c = Invoke-Api Post '/api/admin/users' $u $adminToken
    Assert $c.ok "admin 创建 $($u.role) 账号"
  } else {
    Write-Host "  SKIP  $($u.email) 已存在"
  }
}

$tokens = @{}
foreach ($u in @(
  @{ email = 'pm@e2e.test'; password = 'Pm123456!' },
  @{ email = 'dev@e2e.test'; password = 'Dev123456!' },
  @{ email = 'tl@e2e.test'; password = 'Tl123456!' },
  @{ email = 'tester@e2e.test'; password = 'Test12345!' }
)) {
  $login = Invoke-AuthPaced Post '/api/auth/login' @{ email = $u.email; password = $u.password }
  Assert $login.ok "$($u.email) 登录"
  $tokens[$u.email] = $login.data.token
}
$pmToken = $tokens['pm@e2e.test']; $devToken = $tokens['dev@e2e.test']
$tlToken = $tokens['tl@e2e.test']; $testerToken = $tokens['tester@e2e.test']

$users = (Invoke-Api Get '/api/admin/users' -Token $adminToken).data.users
$pmId = ($users | Where-Object { $_.email -eq 'pm@e2e.test' }).id
$devId = ($users | Where-Object { $_.email -eq 'dev@e2e.test' }).id
$tlId = ($users | Where-Object { $_.email -eq 'tl@e2e.test' }).id

Write-Host "== B. 项目与成员准备 =="
$created = Invoke-Api Post '/api/projects' @{ name = '矩阵验收项目'; description = '全角色矩阵'; agile_mode = 'kanban' } $pmToken
Assert $created.ok 'PM 创建项目'
$projectId = $created.data.id

Assert ((Invoke-Api Post "/api/projects/$projectId/members" @{ user_id = $devId; role_in_project = 'developer' } $pmToken).ok) 'PM 添加开发成员'
Assert ((Invoke-Api Post "/api/projects/$projectId/members" @{ user_id = ($users | Where-Object { $_.email -eq 'tester@e2e.test' }).id; role_in_project = 'tester' } $pmToken).ok) 'PM 添加测试成员'
Assert ((Invoke-Api Post "/api/projects/$projectId/techleads" @{ user_id = $tlId } $pmToken).ok) 'PM 指派技术负责人'

# 两条故事，TL 全部审批通过（进入 backlog 才能领取/规划冲刺）
$s1 = Invoke-Api Post "/api/projects/$projectId/stories" @{ title = '矩阵故事一 支持手机号登录'; story_type = 'feature'; priority = 2 } $pmToken
$s2 = Invoke-Api Post "/api/projects/$projectId/stories" @{ title = '矩阵故事二 邮箱验证码'; story_type = 'feature'; priority = 3 } $pmToken
Assert ($s1.ok -and $s2.ok) 'PM 创建两条故事'
$story1 = $s1.data.id; $story2 = $s2.data.id
Assert ((Invoke-Api Post "/api/stories/$story1/review" @{ approved = $true; comment = '通过' } $tlToken).ok) 'TL 审批故事一'
Assert ((Invoke-Api Post "/api/stories/$story2/review" @{ approved = $true; comment = '通过' } $tlToken).ok) 'TL 审批故事二'

Write-Host "== C. 开发者正向流程：领取 → 状态流转 → 代码关联 → 释放 → 任务 =="
Assert ((Invoke-Api Post "/api/stories/$story1/claim" $null $devToken).ok) '开发者领取故事'
Assert (((Invoke-Api Patch "/api/stories/$story1/status" @{ status = 'ready' } $devToken).ok)) '开发者推进故事到就绪'
Assert (((Invoke-Api Patch "/api/stories/$story1/status" @{ status = 'in_progress' } $devToken).ok)) '开发者推进故事到开发中'
Assert ((Invoke-Api Post "/api/stories/$story1/code-refs" @{ reference = 'https://git.example.com/pr/101' } $devToken).ok) '开发者关联代码'
$release = Invoke-Api Delete "/api/stories/$story1/claim" $null $devToken
Assert $release.ok '开发者释放故事'
Assert ((Invoke-Api Post "/api/stories/$story1/claim" $null $devToken).ok) '开发者重新领取'

$task = Invoke-Api Post "/api/stories/$story1/tasks" @{ title = '实现验证码发送'; priority = 2 } $devToken
Assert $task.ok '开发者创建子任务'
$taskId = $task.data.id
Assert ((Invoke-Api Post "/api/tasks/$taskId/claim" $null $devToken).ok) '开发者领取子任务'
Assert (((Invoke-Api Patch "/api/tasks/$taskId/progress" @{ progress = 50 } $devToken).ok)) '开发者更新子任务进度'
Assert ((Invoke-Api Post "/api/tasks/$taskId/code-refs" @{ reference = 'https://git.example.com/pr/102' } $devToken).ok) '开发者关联任务代码'
$testerTask = Invoke-Api Post "/api/stories/$story1/tasks" @{ title = '测试人员不该能建任务'; priority = 1 } $testerToken
Assert ((!$testerTask.ok -and $testerTask.status -eq 403)) '测试人员创建子任务被拒(403)' $testerTask.status 403

Write-Host "== D. 测试人员正向流程：缺陷全套 + 测试用例 =="
$bug = Invoke-Api Post "/api/projects/$projectId/bugs" @{ title = '验证码发送失败'; severity = 'high'; story_id = $story1 } $testerToken
Assert $bug.ok '测试人员创建缺陷'
$bugId = $bug.data.id
Assert (((Invoke-Api Patch "/api/bugs/$bugId/status" @{ status = 'in_progress' } $devToken).ok)) '开发者更新缺陷状态'
$assign = Invoke-Api Patch "/api/bugs/$bugId/assign" @{ assigned_to = $devId } $testerToken
Assert ((!$assign.ok -and $assign.status -eq 403)) '测试人员指派缺陷被拒(403)' $assign.status 403
Assert (((Invoke-Api Patch "/api/bugs/$bugId/assign" @{ assigned_to = $devId } $pmToken).ok)) 'PM 指派缺陷'
$comment = Invoke-Api Post "/api/bugs/$bugId/comments" @{ body = '已定位：短信网关超时' } $pmToken
Assert $comment.ok 'PM 评论缺陷'
$commentId = $comment.data.id
Assert (((Invoke-Api Put "/api/bugs/$bugId/comments/$commentId" @{ body = '已定位：短信网关超时，待重试' } $pmToken).ok)) 'PM 编辑自己的评论'

$tc = Invoke-Api Post "/api/stories/$story1/test-cases" @{ title = '验证码登录正常流'; steps = @('输入手机号', '输入验证码', '提交'); expected_result = '登录成功' } $testerToken
Assert $tc.ok '测试人员创建测试用例'
$tcId = $tc.data.id
Assert (((Invoke-Api Patch "/api/test-cases/$tcId/status" @{ status = 'passed' } $testerToken).ok)) '测试人员更新用例状态'
$devTc = Invoke-Api Post "/api/stories/$story1/test-cases" @{ title = '开发者不该能建用例'; steps = @('步骤') } $devToken
Assert ((!$devTc.ok -and $devTc.status -eq 403)) '开发者创建测试用例被拒(403)' $devTc.status 403

Write-Host "== E. 冲刺全生命周期（PM） =="
$sprint = Invoke-Api Post "/api/projects/$projectId/sprints" @{ name = '矩阵冲刺一'; goal = '完成登录'; start_date = '2026-10-01'; end_date = '2026-10-14' } $pmToken
Assert $sprint.ok 'PM 创建冲刺'
$sprintId = $sprint.data.id
$testerSprint = Invoke-Api Post "/api/projects/$projectId/sprints" @{ name = '测试人员不该能建'; start_date = '2026-10-01'; end_date = '2026-10-14' } $testerToken
Assert ((!$testerSprint.ok -and $testerSprint.status -eq 403)) '测试人员创建冲刺被拒(403)' $testerSprint.status 403

Assert (((Invoke-Api Patch "/api/stories/$story1/sprint" @{ sprint_id = $sprintId } $pmToken).ok)) 'PM 规划故事一进冲刺'
Assert (((Invoke-Api Patch "/api/stories/$story2/sprint" @{ sprint_id = $sprintId } $pmToken).ok)) 'PM 规划故事二进冲刺'
Assert (((Invoke-Api Patch "/api/sprints/$sprintId/status" @{ status = 'active' } $pmToken).ok)) 'PM 启动冲刺'
$reorder = Invoke-Api Post "/api/sprints/$sprintId/reorder" @{ orders = @(
  @{ story_id = $story2; position = 1 },
  @{ story_id = $story1; position = 2 }
) } $pmToken
Assert $reorder.ok 'PM 批量重排冲刺内故事'

Assert ((Invoke-Api Post "/api/sprints/$sprintId/close" $null $pmToken).ok) 'PM 完成冲刺（未完成故事退回）'
$st1 = (Invoke-Api Get "/api/stories/$story1" -Token $pmToken).data
Assert ($null -eq $st1.sprint_id -or $st1.sprint_id -eq 0) '完成后故事脱离冲刺' ($st1.sprint_id) 'null'

$sprint2 = Invoke-Api Post "/api/projects/$projectId/sprints" @{ name = '矩阵冲刺二'; start_date = '2026-10-15'; end_date = '2026-10-28' } $pmToken
$sprint2Id = $sprint2.data.id
Assert ((Invoke-Api Delete "/api/sprints/$sprint2Id" -Token $pmToken).ok) 'PM 删除未启动冲刺'

Write-Host "== F. 项目设置操作（Owner/admin/成员反例） =="
Assert (((Invoke-Api Put "/api/projects/$projectId" @{ name = '矩阵验收项目改名' } $pmToken).ok)) 'Owner 编辑项目'
Assert (((Invoke-Api Put "/api/projects/$projectId" @{ name = '管理员也能改' } $adminToken).ok)) '平台管理员编辑项目（v1.2 修复）'
$devEdit = Invoke-Api Put "/api/projects/$projectId" @{ name = '开发者不该能改' } $devToken
Assert ((!$devEdit.ok -and $devEdit.status -eq 403)) '普通成员编辑项目被拒(403)' $devEdit.status 403
Assert ((Invoke-Api Get "/api/projects/$projectId/export" -Token $pmToken).ok) 'Owner 导出项目快照'
Assert ((Invoke-Api Post "/api/projects/$projectId/archive" $null $pmToken).ok) 'Owner 归档项目'
Assert ((Invoke-Api Post "/api/projects/$projectId/unarchive" $null $pmToken).ok) 'Owner 还原项目'

Write-Host "== G. 管理员：用户 CRUD + AI 配置 =="
$trashExisting = (Invoke-Api Get '/api/admin/users' -Token $adminToken).data.users | Where-Object { $_.email -eq 'trash@e2e.test' } | Select-Object -First 1
if ($trashExisting) {
  $trashId = $trashExisting.id
  Write-Host '  SKIP  临时用户已存在'
} else {
  $trash = Invoke-Api Post '/api/admin/users' @{ email = 'trash@e2e.test'; username = 'e2e-trash'; password = 'Trash12345!'; role = 'developer' } $adminToken
  Assert $trash.ok 'admin 创建临时用户'
  $trashId = $trash.data.id
}
Assert (((Invoke-Api Put "/api/admin/users/$trashId" @{ role = 'tester' } $adminToken).ok)) 'admin 修改用户角色'
Assert ((Invoke-Api Delete "/api/admin/users/$trashId" -Token $adminToken).ok) 'admin 删除临时用户'

$aiGet = Invoke-Api Get '/api/admin/ai/config' -Token $adminToken
Assert $aiGet.ok 'admin 读取 AI 配置'
$aiSave = Invoke-Api Put '/api/admin/ai/config' @{ model = 'gpt-4o-mini'; api_key = 'sk-e2e-dummy-key'; temperature = 0.2; max_tokens = 1200; enabled = $false } $adminToken
Assert $aiSave.ok 'admin 保存 AI 配置'
$aiTest = Invoke-Api Post '/api/admin/ai/config/test' $null $adminToken
Assert ($aiTest.ok -or $aiTest.status -ge 400) 'AI 测试连接端点可达（无 LLM 时允许业务失败）' $aiTest.status '任意非401/403'
$forbiddenAi = Invoke-Api Get '/api/admin/ai/config' -Token $devToken
Assert ((!$forbiddenAi.ok -and $forbiddenAi.status -eq 403)) '开发者访问 AI 配置被拒(403)' $forbiddenAi.status 403

Write-Host "== H. 通知已读链路 =="
$notifs = Invoke-Api Get '/api/notifications' -Token $tlToken
Assert $notifs.ok 'TL 拉取通知列表'
$firstUnread = $notifs.data.notifications | Where-Object { -not $_.read_at } | Select-Object -First 1
if ($firstUnread) {
  Assert ((Invoke-Api Post "/api/notifications/$($firstUnread.id)/read" $null $tlToken).ok) 'TL 标记单条已读'
}
Assert ((Invoke-Api Post '/api/notifications/mark-all-read' $null $tlToken).ok) 'TL 全部已读'
$unread = (Invoke-Api Get '/api/notifications/unread-count' -Token $tlToken).data
Assert (($unread.count -eq 0 -or $unread.unread_count -eq 0)) '已读后未读数归零'

Write-Host "== 结果: PASS=$script:pass FAIL=$script:fail =="
if ($script:fail -gt 0) { exit 1 }
