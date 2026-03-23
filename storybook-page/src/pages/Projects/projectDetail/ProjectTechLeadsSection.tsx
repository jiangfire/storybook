import Button from '../../../components/ui/Button';
import { CompassIcon } from '../../../components/ui/AppIcon';
import type { User } from '../../../types/models';

interface ProjectTechLeadsSectionProps {
  canManageTechLeads: boolean;
  techLeads: User[];
  availableTechLeadUsers: User[];
  selectedTechLeadUserID: string;
  isAddingTechLead: boolean;
  removingTechLeadID: number | null;
  onSelectedTechLeadUserIDChange: (value: string) => void;
  onAddTechLead: () => void;
  onRemoveTechLead: (userID: number, email?: string) => void;
}

export function ProjectTechLeadsSection({
  canManageTechLeads,
  techLeads,
  availableTechLeadUsers,
  selectedTechLeadUserID,
  isAddingTechLead,
  removingTechLeadID,
  onSelectedTechLeadUserIDChange,
  onAddTechLead,
  onRemoveTechLead,
}: ProjectTechLeadsSectionProps) {
  return (
    <div className="section-card rounded-[1.8rem] p-4 sm:p-5">
      <div className="mb-4">
        <h2 className="text-lg font-semibold text-text">项目技术负责人</h2>
        <p className="mt-1 text-sm text-text-light">明确技术把关角色，减少决策链路里的模糊地带。</p>
      </div>
      {canManageTechLeads && (
        <div className="mb-4 grid grid-cols-1 gap-2 md:grid-cols-2">
          <select
            value={selectedTechLeadUserID}
            onChange={(event) => onSelectedTechLeadUserIDChange(event.target.value)}
            className="field-control"
          >
            <option value="">选择技术负责人</option>
            {availableTechLeadUsers.map((candidate) => (
              <option key={candidate.id} value={candidate.id}>
                {candidate.email}
              </option>
            ))}
          </select>
          <Button size="sm" onClick={onAddTechLead} isLoading={isAddingTechLead}>
            添加技术负责人
          </Button>
        </div>
      )}
      {techLeads.length === 0 ? (
        <div className="state-panel state-panel-empty">当前项目暂无技术负责人</div>
      ) : (
        <div className="space-y-2">
          {techLeads.map((techLead) => (
            <div
              key={techLead.id}
              className="section-block flex flex-col gap-3 rounded-[1.2rem] px-3 py-3 sm:flex-row sm:items-center sm:justify-between"
            >
              <div className="flex items-center gap-2">
                <div className="inline-flex items-center rounded-full bg-amber-50 px-2 py-1 text-amber-700">
                  <CompassIcon size={12} />
                </div>
                <div className="text-sm text-text">{techLead.email}</div>
              </div>
              {canManageTechLeads && (
                <Button
                  size="sm"
                  variant="danger"
                  onClick={() => onRemoveTechLead(techLead.id, techLead.email)}
                  isLoading={removingTechLeadID === techLead.id}
                >
                  移除
                </Button>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
