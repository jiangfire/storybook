import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { aiService } from '../../../services/aiService';
import AIConfigPage from '../AIConfigPage';

const showSuccess = vi.fn();
const showError = vi.fn();

vi.mock('../../../components/ui/Toast', () => ({
  useToast: () => ({
    showSuccess,
    showError,
  }),
}));

vi.mock('../../../services/aiService', () => ({
  aiService: {
    getConfig: vi.fn(),
    updateConfig: vi.fn(),
    testConfig: vi.fn(),
  },
}));

const mockedAIService = vi.mocked(aiService, { deep: true });

describe('AIConfigPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();

    mockedAIService.getConfig.mockResolvedValue({
      config: {
        provider: 'openai',
        model: 'gpt-4.1-mini',
        temperature: 0.3,
        max_tokens: 1500,
        enabled: true,
        api_key_masked: 'sk-***1234',
        is_configured: true,
      },
    });
    mockedAIService.updateConfig.mockResolvedValue({
      config: {
        provider: 'openai',
        model: 'gpt-4.1-mini',
        temperature: 0.5,
        max_tokens: 1800,
        enabled: false,
        api_key_masked: 'sk-***9999',
        is_configured: true,
      },
    });
    mockedAIService.testConfig.mockResolvedValue({
      provider: 'openai',
      model: 'gpt-4.1-mini',
      preview: {
        title: '支持邮箱登录',
        user_story: '作为已注册用户，我希望通过邮箱登录。',
        story_type: 'feature',
        priority: 2,
        story_points: 3,
      },
    });
  });

  it('加载后会展示当前 AI 配置状态', async () => {
    render(<AIConfigPage />);

    await waitFor(() => {
      expect(mockedAIService.getConfig).toHaveBeenCalledTimes(1);
    });

    expect(await screen.findByText('AI 配置')).toBeInTheDocument();
    expect(screen.getByText('已启用 OpenAI')).toBeInTheDocument();
    expect(screen.getByDisplayValue('gpt-4.1-mini')).toBeInTheDocument();
    expect(screen.getByPlaceholderText('sk-***1234')).toBeInTheDocument();
  });

  it('保存时会提交转换后的配置并提示成功', async () => {
    const user = userEvent.setup();

    render(<AIConfigPage />);

    await screen.findByText('AI 配置');
    const spinButtons = screen.getAllByRole('spinbutton');

    await user.clear(screen.getByDisplayValue('gpt-4.1-mini'));
    await user.type(screen.getByPlaceholderText('例如 gpt-4o-mini'), 'gpt-4.1');
    await user.clear(screen.getByDisplayValue('0.3'));
    await user.type(spinButtons[0], '0.5');
    await user.clear(spinButtons[1]);
    await user.type(spinButtons[1], '1800');
    await user.clear(screen.getByPlaceholderText('sk-***1234'));
    await user.type(screen.getByPlaceholderText('sk-***1234'), '  sk-live  ');
    await user.click(screen.getByRole('checkbox'));
    await user.click(screen.getByRole('button', { name: '保存配置' }));

    await waitFor(() => {
      expect(mockedAIService.updateConfig).toHaveBeenCalledWith({
        api_key: 'sk-live',
        model: 'gpt-4.1',
        temperature: 0.5,
        max_tokens: 1800,
        enabled: false,
      });
    });

    expect(showSuccess).toHaveBeenCalledWith('AI 配置已保存');
    expect(screen.getByPlaceholderText('sk-***9999')).toBeInTheDocument();
  });

  it('测试连接成功后会展示预览结果', async () => {
    const user = userEvent.setup();

    render(<AIConfigPage />);

    await screen.findByText('AI 配置');
    await user.click(screen.getByRole('button', { name: '测试连接' }));

    await waitFor(() => {
      expect(mockedAIService.testConfig).toHaveBeenCalledWith({
        api_key: undefined,
        model: 'gpt-4.1-mini',
        temperature: 0.3,
        max_tokens: 1500,
        enabled: true,
      });
    });

    expect(showSuccess).toHaveBeenCalledWith('AI 配置测试通过');
    expect(await screen.findByText('测试结果')).toBeInTheDocument();
    expect(screen.getByText('支持邮箱登录')).toBeInTheDocument();
  });
});
