import { test, expect } from '@playwright/test';

test.describe('Happy Path', () => {
  const testEmail = `e2e-${Date.now()}@example.com`;
  const testPassword = 'Password123';

  test('注册 -> 登录 -> 创建项目 -> 创建故事 -> 查看看板', async ({ page }) => {
    // 1. 注册
    await page.goto('/register');
    await page.fill('input[type="email"]', testEmail);
    await page.fill('input#password', testPassword);
    await page.fill('input#confirmPassword', testPassword);
    await page.click('text=开发人员');
    await page.click('button:has-text("注册")');
    await page.waitForURL('/projects');

    // 2. 创建项目
    await page.click('button:has-text("新建项目")');
    await page.fill('input#new-project-name', 'E2E 测试项目');
    await page.fill('textarea#new-project-description', '由 Playwright E2E 自动创建');
    await page.click('button:has-text("创建项目")');
    await page.waitForURL(/\/projects\/\d+/);

    // 3. 创建故事
    await page.click('text=新建故事');
    await page.fill('input[name="title"]', 'E2E 用户故事');
    await page.fill('textarea[name="description"]', '这是一个端到端测试创建的故事');
    await page.click('button:has-text("创建故事")');

    // 4. 查看看板
    await page.click('text=看板');
    await page.waitForSelector('text=E2E 用户故事');
    await expect(page.locator('text=E2E 用户故事')).toBeVisible();
  });
});
