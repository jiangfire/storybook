import { useState, ChangeEvent } from 'react';
import { aiService } from '../../services/aiService';

interface AICreatorProps {
	onGenerated: (draft: any, strategy: 'replace' | 'fill_empty') => void;
	strategy?: 'replace' | 'fill_empty';
}

export const AICreator = ({ onGenerated, strategy = 'replace' }: AICreatorProps) => {
	const [aiRequirement, setAIRequirement] = useState('');
	const [isGenerating, setIsGenerating] = useState(false);
	const [error, setError] = useState('');

	const handleInputChange = (e: ChangeEvent<HTMLTextAreaElement>) => {
		setAIRequirement(e.target.value);
		if (error) {
			setError('');
		}
	};

	const handleGenerate = async () => {
		const requirement = aiRequirement.trim();
		if (!requirement) {
			setError('请输入需求描述');
			return;
		}

		setIsGenerating(true);
		setError('');

		try {
			const data = await aiService.generateStory({ requirement });
			onGenerated(data.form_draft, strategy);
		} catch (err: unknown) {
			const message = err instanceof Error ? err.message : 'AI生成失败';
			setError(`AI生成失败: ${message}`);
		} finally {
			setIsGenerating(false);
		}
	};

	const handleFillEmpty = async () => {
		// 规则辅助模式 - 使用启发式规则生成草稿
		const draft = {
			title: aiRequirement.trim() || '用户故事',
			description: aiRequirement.trim() || '',
			story_type: 'feature',
			priority: 2,
			acceptance_criteria: [
				' Given 用户未登录',
				' When 用户输入有效的邮箱和密码',
				' Then 用户成功登录并跳转到首页',
			],
			tags: [],
		};

		onGenerated(draft, 'fill_empty');
	};

	const isDisabled = !aiRequirement.trim() || isGenerating;

	return (
		<div className="space-y-6">
			{/* 模型辅助 */}
			<section className="rounded-2xl border border-primary-200 bg-gradient-to-br from-primary-50 via-white to-accent-50 p-5 shadow-sm">
				<div className="flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
					<div className="space-y-1">
						<h3 className="text-lg font-semibold text-text">模型辅助</h3>
						<p className="text-sm text-text-light">
							使用 OpenAI 生成故事草稿，可直接覆盖或只补空白字段
						</p>
					</div>
				</div>

				<div className="mt-5 space-y-3">
					<div>
						<label
							htmlFor="ai-requirement"
							className="mb-2 block text-sm font-medium text-text"
						>
							需求描述
						</label>
						<textarea
							id="ai-requirement"
							value={aiRequirement}
							onChange={handleInputChange}
							rows={4}
							placeholder="输入需求、会议纪要或原话"
							maxLength={2000}
							className="w-full rounded-xl border border-primary-200 bg-white/90 px-3 py-3 text-sm leading-6 text-text shadow-sm outline-none transition focus:border-primary focus:ring-2 focus:ring-primary/20 resize-none"
						/>
					</div>

					{error && <div className="text-xs text-danger">{error}</div>}

					<div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
						<div className="flex flex-wrap gap-2">
							<button
								type="button"
								onClick={handleGenerate}
								disabled={isDisabled}
								className="inline-flex items-center justify-center rounded-xl border border-transparent bg-primary px-4 py-2 text-sm font-medium text-white shadow-sm hover:bg-primary-700 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed"
							>
								{isGenerating ? '生成中...' : '生成草稿'}
							</button>
							<button
								type="button"
								onClick={handleFillEmpty}
								className="inline-flex items-center justify-center rounded-xl border border-gray-300 bg-white px-4 py-2 text-sm font-medium text-text shadow-sm hover:bg-gray-50 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2"
							>
								补空白
							</button>
						</div>
					</div>
				</div>
			</section>

			{/* 规则辅助 */}
			<section className="rounded-2xl border border-amber-200 bg-amber-50/80 p-5 shadow-sm">
				<div className="space-y-1">
					<h3 className="text-lg font-semibold text-text">规则辅助</h3>
					<p className="text-sm text-text-light">
						当未配置 OpenAI、调用失败或返回不可解析时，系统会回退到规则草稿。
					</p>
				</div>

				<div className="mt-4 space-y-3 text-sm text-text-light">
					<div className="rounded-xl border border-amber-200 bg-white/80 px-3 py-3">
						<div className="font-medium text-text">当前兜底能力</div>
						<div className="mt-1">角色、标题、优先级、标签、故事点和基础 AC 会按启发式规则生成。</div>
					</div>
				</div>
			</section>
		</div>
	);
};
