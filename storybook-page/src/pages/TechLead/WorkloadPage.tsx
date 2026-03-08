import { useState, useEffect, useCallback } from 'react';
import { techLeadService } from '../../services/techLeadService';
import type { UserWorkload } from '../../types/models';
import LoadingSpinner from '../../components/ui/LoadingSpinner';

interface ProjectInfo {
  id: number;
  name: string;
  agile_mode: string;
  pending_stories: number;
}

export default function WorkloadPage() {
  const [workloads, setWorkloads] = useState<UserWorkload[]>([]);
  const [projects, setProjects] = useState<ProjectInfo[]>([]);
  const [loading, setLoading] = useState(true);
  const [selectedProject, setSelectedProject] = useState<number | ''>('');

  const loadData = useCallback(async () => {
    try {
      setLoading(true);
      const [workloadRes, projectsRes] = await Promise.all([
        techLeadService.getWorkload(selectedProject || undefined),
        techLeadService.getMyProjects(),
      ]);
      setWorkloads(workloadRes.data.workloads);
      setProjects(projectsRes.data.projects);
    } catch (error) {
      console.error('Failed to load workload:', error);
    } finally {
      setLoading(false);
    }
  }, [selectedProject]);

  useEffect(() => {
    loadData();
  }, [loadData]);

  const getWorkloadLevel = (workload: UserWorkload) => {
    const score = workload.total_story_points + workload.estimated_hours / 4;
    if (score > 20) return { level: 'high', color: 'bg-red-100 text-red-800', label: '高负载' };
    if (score > 10)
      return { level: 'medium', color: 'bg-yellow-100 text-yellow-800', label: '中负载' };
    return { level: 'low', color: 'bg-green-100 text-green-800', label: '低负载' };
  };

  const getCompletionRateColor = (rate: number) => {
    if (rate >= 80) return 'text-green-600';
    if (rate >= 50) return 'text-yellow-600';
    return 'text-red-600';
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <LoadingSpinner />
      </div>
    );
  }

  return (
    <div className="container mx-auto px-4 py-6">
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-text mb-2">工作负载</h1>
        <p className="text-text-light">查看团队成员的工作负载情况</p>
      </div>

      {/* 筛选栏 */}
      <div className="bg-white rounded-lg shadow-sm border border-border p-4 mb-6">
        <div className="flex flex-wrap gap-4">
          <div className="w-64">
            <label className="block text-sm font-medium text-text mb-1">项目</label>
            <select
              value={selectedProject}
              onChange={(e) => setSelectedProject(e.target.value ? Number(e.target.value) : '')}
              className="w-full px-3 py-2 border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-primary-500"
            >
              <option value="">全部项目</option>
              {projects.map((project) => (
                <option key={project.id} value={project.id}>
                  {project.name}
                </option>
              ))}
            </select>
          </div>
        </div>
      </div>

      {/* 统计概览 */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <div className="bg-white rounded-lg shadow-sm border border-border p-4">
          <div className="text-2xl font-bold text-text">{workloads.length}</div>
          <div className="text-sm text-text-light">团队成员</div>
        </div>
        <div className="bg-white rounded-lg shadow-sm border border-border p-4">
          <div className="text-2xl font-bold text-text">
            {workloads.reduce((sum, w) => sum + w.active_stories, 0)}
          </div>
          <div className="text-sm text-text-light">活跃故事</div>
        </div>
        <div className="bg-white rounded-lg shadow-sm border border-border p-4">
          <div className="text-2xl font-bold text-text">
            {workloads.reduce((sum, w) => sum + w.total_story_points, 0)}
          </div>
          <div className="text-sm text-text-light">总故事点</div>
        </div>
        <div className="bg-white rounded-lg shadow-sm border border-border p-4">
          <div className="text-2xl font-bold text-text">
            {workloads.reduce((sum, w) => sum + w.estimated_hours, 0).toFixed(1)}
          </div>
          <div className="text-sm text-text-light">预估工时</div>
        </div>
      </div>

      {/* 负载详情表格 */}
      <div className="bg-white rounded-lg shadow-sm border border-border overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead className="bg-gray-50 border-b border-border">
              <tr>
                <th className="px-4 py-3 text-left text-sm font-medium text-text">成员</th>
                <th className="px-4 py-3 text-center text-sm font-medium text-text">负载状态</th>
                <th className="px-4 py-3 text-center text-sm font-medium text-text">活跃故事</th>
                <th className="px-4 py-3 text-center text-sm font-medium text-text">活跃任务</th>
                <th className="px-4 py-3 text-center text-sm font-medium text-text">故事点</th>
                <th className="px-4 py-3 text-center text-sm font-medium text-text">预估工时</th>
                <th className="px-4 py-3 text-center text-sm font-medium text-text">完成率</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {workloads.length === 0 ? (
                <tr>
                  <td colSpan={7} className="px-4 py-8 text-center text-text-light">
                    暂无数据
                  </td>
                </tr>
              ) : (
                workloads.map((workload) => {
                  const workloadInfo = getWorkloadLevel(workload);
                  return (
                    <tr key={workload.user.id} className="hover:bg-gray-50">
                      <td className="px-4 py-3">
                        <div className="flex items-center gap-3">
                          <div className="w-8 h-8 rounded-full bg-primary-100 flex items-center justify-center text-primary font-medium">
                            {workload.user.email.charAt(0).toUpperCase()}
                          </div>
                          <div>
                            <div className="font-medium text-text">{workload.user.email}</div>
                            <div className="text-xs text-text-light">ID: {workload.user.id}</div>
                          </div>
                        </div>
                      </td>
                      <td className="px-4 py-3 text-center">
                        <span
                          className={`px-2 py-1 rounded-full text-xs font-medium ${workloadInfo.color}`}
                        >
                          {workloadInfo.label}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-center text-text">{workload.active_stories}</td>
                      <td className="px-4 py-3 text-center text-text">{workload.active_tasks}</td>
                      <td className="px-4 py-3 text-center text-text">
                        {workload.total_story_points}
                      </td>
                      <td className="px-4 py-3 text-center text-text">
                        {workload.estimated_hours.toFixed(1)}h
                      </td>
                      <td className="px-4 py-3 text-center">
                        <span
                          className={`font-medium ${getCompletionRateColor(workload.completion_rate)}`}
                        >
                          {workload.completion_rate.toFixed(1)}%
                        </span>
                      </td>
                    </tr>
                  );
                })
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
