import { create } from 'zustand';
import { Project, ProjectOverview } from '../types/models';
import { ProjectListParams, CreateProjectRequest } from '../types/api';
import { projectService } from '../services/projectService';
import { getErrorMessage } from '../utils/error';

interface ProjectState {
  currentProject: Project | null;
  projects: Project[];
  projectOverview: ProjectOverview | null;
  isLoading: boolean;
  error: string | null;

  // Actions
  fetchProjects: (params?: ProjectListParams) => Promise<void>;
  fetchProject: (id: number) => Promise<void>;
  fetchProjectOverview: (id: number) => Promise<void>;
  createProject: (data: CreateProjectRequest) => Promise<Project>;
  updateProject: (id: number, data: Partial<CreateProjectRequest>) => Promise<void>;
  deleteProject: (id: number) => Promise<void>;
  setCurrentProject: (project: Project | null) => void;
  clearError: () => void;
}

export const useProjectStore = create<ProjectState>((set) => ({
  currentProject: null,
  projects: [],
  projectOverview: null,
  isLoading: false,
  error: null,

  fetchProjects: async (params) => {
    set({ isLoading: true, error: null });
    try {
      const response = await projectService.getProjects(params);
      set({ projects: response.projects, isLoading: false });
    } catch (error: unknown) {
      set({ error: getErrorMessage(error), isLoading: false });
    }
  },

  fetchProject: async (id) => {
    set({ isLoading: true, error: null });
    try {
      const project = await projectService.getProject(id);
      set({ currentProject: project, isLoading: false });
    } catch (error: unknown) {
      set({ error: getErrorMessage(error), isLoading: false });
    }
  },

  fetchProjectOverview: async (id) => {
    set({ isLoading: true, error: null });
    try {
      const overview = await projectService.getProjectOverview(id);
      set({ projectOverview: overview, isLoading: false });
    } catch (error: unknown) {
      set({ error: getErrorMessage(error), isLoading: false });
    }
  },

  createProject: async (data) => {
    set({ isLoading: true, error: null });
    try {
      const project = await projectService.createProject(data);
      set((state) => ({
        projects: [project, ...state.projects],
        isLoading: false,
      }));
      return project;
    } catch (error: unknown) {
      set({ error: getErrorMessage(error), isLoading: false });
      throw error;
    }
  },

  updateProject: async (id, data) => {
    set({ isLoading: true, error: null });
    try {
      const updatedProject = await projectService.updateProject(id, data);
      set((state) => ({
        projects: state.projects.map((p) => (p.id === id ? updatedProject : p)),
        currentProject: state.currentProject?.id === id ? updatedProject : state.currentProject,
        isLoading: false,
      }));
    } catch (error: unknown) {
      set({ error: getErrorMessage(error), isLoading: false });
    }
  },

  deleteProject: async (id) => {
    set({ isLoading: true, error: null });
    try {
      await projectService.deleteProject(id);
      set((state) => ({
        projects: state.projects.filter((p) => p.id !== id),
        currentProject: state.currentProject?.id === id ? null : state.currentProject,
        isLoading: false,
      }));
    } catch (error: unknown) {
      set({ error: getErrorMessage(error), isLoading: false });
    }
  },

  setCurrentProject: (project) => {
    set({ currentProject: project });
  },

  clearError: () => set({ error: null }),
}));
