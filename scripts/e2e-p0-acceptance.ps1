# Storybook P0 闭环 E2E 验收脚本（API 级）
# 用法: pwsh -File e2e.ps1  (需要先启动服务并设置 $env:E2E_BASE)
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

Write-Host "== 1. 登录与账号准备（认证接口限流 3次/分，逐个加间隔） =="

# 登录/注册接口有 IP 限流（3次/分，burst=1），认证调用之间必须留出间隔
function Invoke-AuthPaced {
  param([string]$Method, [string]$Path, $Body)
  Start-Sleep -Seconds 30
  return Invoke-Api -Method $Method -Path $Path -Body $Body
}

$admin = Invoke-AuthPaced Post '/api/auth/login' @{ email = 'admin@e2e.test'; password = 'Admin12345!' }
Assert $admin.ok 'admin 登录'
$adminToken = $admin.data.token

$users = (Invoke-Api Get '/api/admin/users' -Token $adminToken).data
$tlExisting = $users.users | Where-Object { $_.email -eq 'tl@e2e.test' } | Select-Object -First 1
if (-not $tlExisting) {
  $tl = Invoke-Api Post '/api/admin/users' @{ email = 'tl@e2e.test'; username = 'e2e-techlead'; password = 'Tl123456!'; role = 'tech_lead' } $adminToken
  Assert $tl.ok 'admin 创建技术负责人账号'
} else {
  Write-Host '  SKIP  技术负责人账号已存在'
}
$tlLogin = Invoke-AuthPaced Post '/api/auth/login' @{ email = 'tl@e2e.test'; password = 'Tl123456!' }
Assert $tlLogin.ok '技术负责人登录'
$tlToken = $tlLogin.data.token

$pmLogin = Invoke-AuthPaced Post '/api/auth/login' @{ email = 'pm@e2e.test'; password = 'Pm123456!' }
if (-not $pmLogin.ok) {
  $reg = Invoke-AuthPaced Post '/api/auth/register' @{ email = 'pm@e2e.test'; password = 'Pm123456!'; role = 'product' }
  Assert $reg.ok '注册产品经理'
  $pmLogin = Invoke-AuthPaced Post '/api/auth/login' @{ email = 'pm@e2e.test'; password = 'Pm123456!' }
} else {
  Write-Host '  SKIP  产品经理已存在，直接登录'
}
Assert $pmLogin.ok '产品经理登录'
$pmToken = $pmLogin.data.token

$devLogin = Invoke-AuthPaced Post '/api/auth/login' @{ email = 'dev@e2e.test'; password = 'Dev123456!' }
if (-not $devLogin.ok) {
  $reg = Invoke-AuthPaced Post '/api/auth/register' @{ email = 'dev@e2e.test'; password = 'Dev123456!'; role = 'developer' }
  Assert $reg.ok '注册开发者'
  $devId = $reg.data.user.id
  $devLogin = Invoke-AuthPaced Post '/api/auth/login' @{ email = 'dev@e2e.test'; password = 'Dev123456!' }
} else {
  Write-Host '  SKIP  开发者已存在，直接登录'
}
Assert $devLogin.ok '开发者登录'
$devToken = $devLogin.data.token

$users = (Invoke-Api Get '/api/admin/users' -Token $adminToken).data
$devId = ($users.users | Where-Object { $_.email -eq 'dev@e2e.test' } | Select-Object -First 1).id
$tlId = ($users.users | Where-Object { $_.email -eq 'tl@e2e.test' } | Select-Object -First 1).id
Assert ([bool]$tlId) "拿到技术负责人 user_id (tlId=$tlId)"

Write-Host "== 2. 项目与 PM 指定技术负责人 =="
$myProjects = (Invoke-Api Get '/api/projects' -Token $pmToken).data.projects
$proj = $myProjects | Where-Object { $_.name -eq 'E2E 验收项目' } | Select-Object -First 1
if (-not $proj) {
  $created = Invoke-Api Post '/api/projects' @{ name = 'E2E 验收项目'; description = 'P0 闭环验收'; agile_mode = 'kanban' } $pmToken
  Assert $created.ok 'PM 创建项目'
  $projectId = $created.data.id
  if (-not $projectId) { $projectId = ((Invoke-Api Get '/api/projects' -Token $pmToken).data.projects | Select-Object -First 1).id }
} else {
  Write-Host '  SKIP  项目已存在，复用'
  $projectId = $proj.id
}
Assert ([bool]$projectId) "拿到项目 id (projectId=$projectId)"

$tlList = (Invoke-Api Get "/api/projects/$projectId/techleads" -Token $pmToken).data.tech_leads
if (-not ($tlList | Where-Object { $_.id -eq $tlId })) {
  $addTl = Invoke-Api Post "/api/projects/$projectId/techleads" @{ user_id = $tlId } $pmToken
  Assert $addTl.ok 'PM 可以为项目指定技术负责人（新权限）'
} else {
  Write-Host '  SKIP  技术负责人已在项目中'
}

$tlList = (Invoke-Api Get "/api/projects/$projectId/techleads" -Token $pmToken).data.tech_leads
Assert (($tlList | Where-Object { $_.id -eq $tlId }) -ne $null) '技术负责人出现在项目技术负责人列表'

Write-Host "== 3. 创建故事 → 审批通知 → 催审 =="
$story = Invoke-Api Post "/api/projects/$projectId/stories" @{ title = '支持手机号验证码登录'; story_type = 'feature'; priority = 3 } $pmToken
Assert $story.ok 'PM 创建故事'
$storyId = $story.data.id
Assert ($story.data.status -eq 'pending') '新故事状态为 pending' $story.data.status 'pending'

$tlNotif = (Invoke-Api Get '/api/notifications' -Token $tlToken).data.items
Assert (($tlNotif | Where-Object { $_.type -eq 'story.review_requested' -and $_.entity_id -eq $storyId }) -ne $null) '技术负责人收到审批请求通知'

$pending = (Invoke-Api Get '/api/techlead/pending-stories' -Token $tlToken).data
Assert ((($pending.stories | Where-Object { $_.id -eq $storyId }) -ne $null)) '待审批列表包含新故事'

$urge = Invoke-Api Post "/api/stories/$storyId/urge-review" -Token $pmToken
Assert $urge.ok 'PM 催审成功'
$tlNotif2 = (Invoke-Api Get '/api/notifications' -Token $tlToken).data.items
Assert (($tlNotif2 | Where-Object { $_.type -eq 'story.review_urged' -and $_.entity_id -eq $storyId }) -ne $null) '技术负责人收到催审通知'

Write-Host "== 4. 驳回 → 重提 → 通过 =="
$reject = Invoke-Api Post "/api/stories/$storyId/review" @{ approved = $false; comment = 'AC 不可验证' } $tlToken
Assert $reject.ok '技术负责人驳回并填写原因'

$pmNotif = (Invoke-Api Get '/api/notifications' -Token $pmToken).data.items
Assert (($pmNotif | Where-Object { $_.type -eq 'story.reviewed' -and $_.entity_id -eq $storyId }) -ne $null) '产品经理收到审批结果通知'

$detail = (Invoke-Api Get "/api/stories/$storyId" -Token $pmToken).data
Assert ($detail.review_status -eq 'rejected') '故事处于被驳回状态' $detail.review_status 'rejected'
Assert ([bool]$detail.reviewed_at) '详情接口返回审批时间'

$resub = Invoke-Api Post "/api/stories/$storyId/resubmit" -Token $pmToken
Assert $resub.ok 'PM 重新提交审批'
$detail = (Invoke-Api Get "/api/stories/$storyId" -Token $pmToken).data
Assert ($detail.review_status -eq 'pending') '重提后 review_status 复位 pending' $detail.review_status 'pending'

$tlNotif3 = (Invoke-Api Get '/api/notifications' -Token $tlToken).data.items
Assert (($tlNotif3 | Where-Object { $_.type -eq 'story.review_resubmitted' -and $_.entity_id -eq $storyId }) -ne $null) '技术负责人收到重提通知'

$approve = Invoke-Api Post "/api/stories/$storyId/review" @{ approved = $true } $tlToken
Assert $approve.ok '技术负责人审批通过'
$detail = (Invoke-Api Get "/api/stories/$storyId" -Token $pmToken).data
Assert ($detail.status -eq 'backlog') '通过后故事进入 backlog' $detail.status 'backlog'

Write-Host "== 5. 权限反例 =="
$devResub = Invoke-Api Post "/api/stories/$storyId/resubmit" -Token $devToken
Assert ($devResub.ok -eq $false) '开发者重提被拒（非项目成员/无权限）' $devResub.status '40x'

$members = (Invoke-Api Get "/api/projects/$projectId/members" -Token $pmToken).data.members
if (-not ($members | Where-Object { $_.user_id -eq $devId })) {
  $pmAddDev = Invoke-Api Post "/api/projects/$projectId/members" @{ user_id = $devId; role_in_project = 'developer' } $pmToken
  Assert $pmAddDev.ok 'PM 添加开发成员'
} else {
  Write-Host '  SKIP  开发成员已在项目中'
}

$devUrge = Invoke-Api Post "/api/stories/$storyId/urge-review" -Token $devToken
Assert ((-not $devUrge.ok) -and $devUrge.status -eq 403) '开发者催审被拒(403)' $devUrge.status 403

$devAddTl = Invoke-Api Post "/api/projects/$projectId/techleads" @{ user_id = $tlId } $devToken
Assert ((-not $devAddTl.ok) -and $devAddTl.status -eq 403) '开发者（项目成员）指定技术负责人被拒(403)' $devAddTl.status 403

$devReject = Invoke-Api Post "/api/stories/$storyId/review" @{ approved = $true } $devToken
Assert ($devReject.ok -eq $false) '开发者审批被拒' $devReject.status '403/400'

$anon = Invoke-Api Get '/api/projects'
Assert ((-not $anon.ok) -and $anon.status -eq 401) '未登录访问被拒(401)' $anon.status 401

Write-Host "== 结果: PASS=$($script:pass) FAIL=$($script:fail) =="
if ($script:fail -gt 0) { exit 1 } else { exit 0 }
