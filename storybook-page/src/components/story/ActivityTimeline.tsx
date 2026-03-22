import type { Activity } from '../../types/models';
import { formatRelativeTime, getUserInitials } from '../../utils/formatters';
import { cn } from '../../utils/cn';
import {
  ClipboardIcon,
  InboxIcon,
  SparklesIcon,
  UsersIcon,
  WrenchIcon,
} from '../ui/AppIcon';

interface ActivityTimelineProps {
  activities: Activity[];
}

const actionConfig: Record<
  string,
  {
    label: string;
    icon: typeof SparklesIcon;
    bgColor: string;
    iconColor: string;
  }
> = {
  created: { label: '创建了', icon: SparklesIcon, bgColor: 'bg-blue-50', iconColor: 'text-blue-600' },
  updated: { label: '更新了', icon: WrenchIcon, bgColor: 'bg-yellow-50', iconColor: 'text-yellow-600' },
  status_changed: {
    label: '状态变更为',
    icon: ClipboardIcon,
    bgColor: 'bg-purple-50',
    iconColor: 'text-purple-600',
  },
  assigned: { label: '分配给', icon: UsersIcon, bgColor: 'bg-green-50', iconColor: 'text-green-600' },
  commented: {
    label: '评论了',
    icon: ClipboardIcon,
    bgColor: 'bg-secondary-50',
    iconColor: 'text-text-light',
  },
};

export default function ActivityTimeline({ activities }: ActivityTimelineProps) {
  if (activities.length === 0) {
    return (
      <div className="text-center py-8 text-text-light">
        <div className="mb-2 inline-flex h-12 w-12 items-center justify-center rounded-full bg-secondary-50">
          <InboxIcon size={20} />
        </div>
        <p>暂无活动记录</p>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {activities.map((activity, index) => {
        const config = actionConfig[activity.action] || actionConfig.updated;
        const Icon = config.icon;

        return (
          <div key={activity.id} className="flex space-x-3">
            {/* 时间轴线 */}
            <div className="flex flex-col items-center">
              {/* 图标 */}
              <div
                className={cn(
                  'w-8 h-8 rounded-full flex items-center justify-center text-sm flex-shrink-0',
                  config.bgColor
                )}
              >
                <Icon size={14} className={config.iconColor} />
              </div>
              {/* 连接线 */}
              {index < activities.length - 1 && (
                <div className="w-0.5 flex-1 bg-secondary-200 my-1 min-h-[2rem]" />
              )}
            </div>

            {/* 内容 */}
            <div className="flex-1 pb-4">
              <div className="bg-white rounded-lg border border-border p-4">
                {/* 操作信息 */}
                <div className="flex items-start justify-between mb-2">
                  <div className="flex items-center space-x-2">
                    {/* 用户头像 */}
                    <div
                      className="w-6 h-6 rounded-full bg-primary-100 text-primary flex items-center justify-center text-xs font-medium"
                      title={activity.user.email}
                    >
                      {getUserInitials(activity.user.email)}
                    </div>
                    <span className="text-sm font-medium text-text">{activity.user.email}</span>
                    <span className="text-sm text-text-light">{config.label}</span>
                  </div>
                  <span className="text-xs text-text-light">
                    {formatRelativeTime(activity.created_at)}
                  </span>
                </div>

                {/* 变更详情 */}
                {(activity.old_value || activity.new_value) && (
                  <div className="mt-3 space-y-2">
                    {activity.old_value && (
                      <div className="bg-red-50 rounded p-2 text-sm">
                        <div className="text-xs text-red-700 mb-1">之前:</div>
                        <div className="text-red-900">
                          {Object.entries(activity.old_value).map(([key, value]) => (
                            <div key={key}>
                              <span className="font-medium">{key}:</span>{' '}
                              <span>{String(value)}</span>
                            </div>
                          ))}
                        </div>
                      </div>
                    )}
                    {activity.new_value && (
                      <div className="bg-green-50 rounded p-2 text-sm">
                        <div className="text-xs text-green-700 mb-1">之后:</div>
                        <div className="text-green-900">
                          {Object.entries(activity.new_value).map(([key, value]) => (
                            <div key={key}>
                              <span className="font-medium">{key}:</span>{' '}
                              <span>{String(value)}</span>
                            </div>
                          ))}
                        </div>
                      </div>
                    )}
                  </div>
                )}
              </div>
            </div>
          </div>
        );
      })}
    </div>
  );
}
