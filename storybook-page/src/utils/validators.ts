/**
 * 验证邮箱格式
 */
export function isValidEmail(email: string): boolean {
  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
  return emailRegex.test(email);
}

/**
 * 验证密码强度
 * 至少8位，包含字母和数字
 */
export function isValidPassword(password: string): boolean {
  if (password.length < 8) return false;
  const hasLetter = /[a-zA-Z]/.test(password);
  const hasNumber = /[0-9]/.test(password);
  return hasLetter && hasNumber;
}

/**
 * 验证项目名称
 */
export function isValidProjectName(name: string): boolean {
  return name.length >= 2 && name.length <= 100;
}

/**
 * 验证故事标题
 */
export function isValidStoryTitle(title: string): boolean {
  return title.length >= 2 && title.length <= 200;
}

/**
 * 验证故事描述
 */
export function isValidStoryDescription(description: string): boolean {
  return description.length <= 2000;
}

/**
 * 验证URL格式
 */
export function isValidURL(url: string): boolean {
  try {
    new URL(url);
    return true;
  } catch {
    return false;
  }
}

/**
 * 验证文件路径
 */
export function isValidFilePath(path: string): boolean {
  // 简单的文件路径验证
  const pathRegex = /^[\w\-./\\]+$/;
  return pathRegex.test(path);
}
