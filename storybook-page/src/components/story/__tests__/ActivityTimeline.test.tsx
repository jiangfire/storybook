import { render, screen } from '@testing-library/react';
import type { Activity } from '../../../types/models';
import ActivityTimeline from '../ActivityTimeline';

const NOW = '2026-03-29T00:00:00Z';

function createActivity(id: number, overrides: Partial<Activity> = {}): Activity {
  return {
    id,
    entity_type: 'story',
    entity_id: 18,
    action: 'created',
    user: {
      id: 1,
      email: 'owner@example.com',
      role: 'product',
      created_at: NOW,
    },
    created_at: NOW,
    ...overrides,
  };
}

describe('ActivityTimeline', () => {
  it('空活动列表显示空状态', () => {
    render(<ActivityTimeline activities={[]} />);

    expect(screen.getByText('暂无活动记录')).toBeInTheDocument();
  });

  it('展示活动详情与前后变更', () => {
    render(
      <ActivityTimeline
        activities={[
          createActivity(1, {
            action: 'updated',
            old_value: { status: 'pending' },
            new_value: { status: 'ready' },
          }),
          createActivity(2, {
            action: 'unknown_action',
            new_value: { assignee: 'dev@example.com' },
          }),
        ]}
      />
    );

    expect(screen.getAllByText('owner@example.com').length).toBeGreaterThan(0);
    expect(screen.getAllByText('更新了').length).toBeGreaterThan(0);
    expect(screen.getByText('之前:')).toBeInTheDocument();
    expect(screen.getAllByText('之后:').length).toBeGreaterThan(0);
    expect(screen.getAllByText('status:').length).toBeGreaterThan(0);
    expect(screen.getByText('ready')).toBeInTheDocument();
  });
});
