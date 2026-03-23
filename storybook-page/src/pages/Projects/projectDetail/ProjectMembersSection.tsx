import Button from '../../../components/ui/Button';
import {
  ClipboardIcon,
  CodeIcon,
  CrownIcon,
  SearchIcon,
} from '../../../components/ui/AppIcon';
import type { ProjectRole, User } from '../../../types/models';
import type { ProjectMemberItem } from './types';

const memberRoleMeta: Record<
  string,
  { icon: typeof ClipboardIcon; label: string; colorClass: string; bgClass: string }
> = {
  product: {
    icon: ClipboardIcon,
    label: '产品经理',
    colorClass: 'text-blue-700',
    bgClass: 'bg-blue-50',
  },
  developer: {
    icon: CodeIcon,
    label: '开发',
    colorClass: 'text-emerald-700',
    bgClass: 'bg-emerald-50',
  },
  tester: {
    icon: SearchIcon,
    label: '测试',
    colorClass: 'text-violet-700',
    bgClass: 'bg-violet-50',
  },
};

function getMemberRoleMeta(role: string) {
  return (
    memberRoleMeta[role] || {
      icon: SearchIcon,
      label: role,
      colorClass: 'text-text',
      bgClass: 'bg-secondary-50',
    }
  );
}

interface ProjectMembersSectionProps {
  canManageMembers: boolean;
  memberError: string;
  members: ProjectMemberItem[];
  availableMemberUsers: User[];
  selectedMemberUserID: string;
  selectedMemberRole: ProjectRole;
  isAddingMember: boolean;
  removingMemberUserID: number | null;
  onSelectedMemberUserIDChange: (value: string) => void;
  onSelectedMemberRoleChange: (role: ProjectRole) => void;
  onAddMember: () => void;
  onRemoveMember: (member: ProjectMemberItem) => void;
}

export function ProjectMembersSection({
  canManageMembers,
  memberError,
  members,
  availableMemberUsers,
  selectedMemberUserID,
  selectedMemberRole,
  isAddingMember,
  removingMemberUserID,
  onSelectedMemberUserIDChange,
  onSelectedMemberRoleChange,
  onAddMember,
  onRemoveMember,
}: ProjectMembersSectionProps) {
  return (
    <div className="section-card rounded-[1.8rem] p-4 sm:p-5">
      <div className="mb-4 flex flex-col gap-2">
        <h2 className="text-lg font-semibold text-text">项目成员</h2>
        <div className="flex flex-wrap items-center gap-2 text-xs text-text-light">
          <span className="inline-flex items-center gap-1 rounded-full bg-blue-50 px-2 py-1 text-blue-700">
            <ClipboardIcon size={12} />
            产品
          </span>
          <span className="inline-flex items-center gap-1 rounded-full bg-emerald-50 px-2 py-1 text-emerald-700">
            <CodeIcon size={12} />
            开发
          </span>
          <span className="inline-flex items-center gap-1 rounded-full bg-violet-50 px-2 py-1 text-violet-700">
            <SearchIcon size={12} />
            测试
          </span>
          <span className="inline-flex items-center gap-1 rounded-full bg-amber-50 px-2 py-1 text-amber-700">
            <CrownIcon size={12} />
            负责人
          </span>
        </div>
      </div>
      {canManageMembers && (
        <div className="mb-4 grid grid-cols-1 gap-2 md:grid-cols-3">
          <select
            value={selectedMemberUserID}
            onChange={(event) => onSelectedMemberUserIDChange(event.target.value)}
            className="field-control"
          >
            <option value="">选择用户</option>
            {availableMemberUsers.map((candidate) => (
              <option key={candidate.id} value={candidate.id}>
                {candidate.email}（{candidate.role}）
              </option>
            ))}
          </select>
          <select
            value={selectedMemberRole}
            onChange={(event) => onSelectedMemberRoleChange(event.target.value as ProjectRole)}
            className="field-control"
          >
            <option value="product">产品经理</option>
            <option value="developer">开发</option>
            <option value="tester">测试</option>
          </select>
          <Button size="sm" onClick={onAddMember} isLoading={isAddingMember}>
            添加成员
          </Button>
        </div>
      )}
      {memberError ? (
        <div className="state-panel state-panel-error">{memberError}</div>
      ) : members.length === 0 ? (
        <div className="state-panel state-panel-empty">暂无成员</div>
      ) : (
        <div className="space-y-2">
          {members.map((member) => {
            const roleMeta = getMemberRoleMeta(member.role_in_project);
            const RoleIcon = roleMeta.icon;

            return (
              <div
                key={member.id}
                className="section-block flex flex-col gap-3 rounded-[1.2rem] px-3 py-3 sm:flex-row sm:items-center sm:justify-between"
              >
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-2">
                    <div className="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-full bg-primary-100 text-sm font-medium text-primary">
                      {(member.user?.email || `#${member.user_id}`).charAt(0).toUpperCase()}
                    </div>
                    <div className="min-w-0">
                      <div className="truncate text-sm font-medium text-text">
                        {member.user?.email || `用户 #${member.user_id}`}
                      </div>
                      <div className="mt-1 flex flex-wrap items-center gap-2 text-xs text-text-light">
                        <span
                          className={`inline-flex items-center rounded-full px-2 py-1 ${roleMeta.bgClass} ${roleMeta.colorClass}`}
                          title={roleMeta.label}
                          aria-label={roleMeta.label}
                        >
                          <RoleIcon size={12} />
                        </span>
                        {member.is_owner && (
                          <span
                            className="inline-flex items-center rounded-full bg-amber-50 px-2 py-1 text-amber-700"
                            title="项目负责人"
                            aria-label="项目负责人"
                          >
                            <CrownIcon size={12} />
                          </span>
                        )}
                        <span>{new Date(member.joined_at).toLocaleDateString()}</span>
                      </div>
                    </div>
                  </div>
                </div>
                <div className="flex items-center justify-end gap-2">
                  {canManageMembers && !member.is_owner && (
                    <Button
                      size="sm"
                      variant="danger"
                      onClick={() => onRemoveMember(member)}
                      isLoading={removingMemberUserID === member.user_id}
                    >
                      移除
                    </Button>
                  )}
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
