import {
  isValidEmail,
  isValidFilePath,
  isValidPassword,
  isValidProjectName,
  isValidStoryDescription,
  isValidStoryTitle,
  isValidURL,
} from '../validators';

describe('validators', () => {
  it('验证邮箱格式', () => {
    expect(isValidEmail('user@example.com')).toBe(true);
    expect(isValidEmail('bad-email')).toBe(false);
  });

  it('验证密码强度', () => {
    expect(isValidPassword('Pass1234')).toBe(true);
    expect(isValidPassword('short1')).toBe(false);
    expect(isValidPassword('allletters')).toBe(false);
    expect(isValidPassword('12345678')).toBe(false);
  });

  it('验证项目名称和故事标题长度', () => {
    expect(isValidProjectName('AB')).toBe(true);
    expect(isValidProjectName('A')).toBe(false);
    expect(isValidStoryTitle('功能A')).toBe(true);
    expect(isValidStoryTitle('x')).toBe(false);
  });

  it('验证故事描述长度', () => {
    expect(isValidStoryDescription('ok')).toBe(true);
    expect(isValidStoryDescription('a'.repeat(2001))).toBe(false);
  });

  it('验证 URL 与文件路径', () => {
    expect(isValidURL('https://example.com/a')).toBe(true);
    expect(isValidURL('not-url')).toBe(false);
    expect(isValidFilePath('src/components/Button.tsx')).toBe(true);
    expect(isValidFilePath('bad:path<>')).toBe(false);
  });
});
