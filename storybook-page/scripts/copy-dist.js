#!/usr/bin/env node
/**
 * 复制前端构建产物到后端嵌入目录
 * 用于单体部署时嵌入前端资源
 */

import { existsSync, mkdirSync, cpSync, rmSync } from 'fs';
import { join, dirname } from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

const projectRoot = join(__dirname, '..', '..');
const sourceDir = join(projectRoot, 'storybook-page', 'dist');
const targetDir = join(projectRoot, 'internal', 'webui', 'dist');

// 检查源目录是否存在
if (!existsSync(sourceDir)) {
  console.error('❌ 前端构建产物不存在:', sourceDir);
  console.error('请先运行 pnpm build');
  process.exit(1);
}

// 确保目标目录存在
if (existsSync(targetDir)) {
  // 清空旧内容
  rmSync(targetDir, { recursive: true });
}
mkdirSync(targetDir, { recursive: true });

// 复制文件
try {
  cpSync(sourceDir, targetDir, { recursive: true });
  console.log('✅ 前端构建产物已复制到:', targetDir);
  console.log('📦 可以运行 go build 构建后端了');
} catch (err) {
  console.error('❌ 复制失败:', err.message);
  process.exit(1);
}
