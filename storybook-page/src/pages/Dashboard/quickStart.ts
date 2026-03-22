import type { Project } from '../../types/models';

export function resolveQuickStartProject(
  projects: Project[],
  selectedProjectId: number | null
): Project | null {
  if (!projects || projects.length === 0) {
    return null;
  }

  if (selectedProjectId !== null) {
    const selectedProject = projects.find((project) => project.id === selectedProjectId);
    if (selectedProject) {
      return selectedProject;
    }
  }

  return projects[0];
}
