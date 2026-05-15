import { http, HttpResponse } from 'msw';

// In-memory fixtures so multiple requests can share state during a single test.
let currentUser: {
  id: number;
  email: string;
  role: string;
  username: string;
} | null = null;

let nextProjectId = 1;
const projects: Array<{
  id: number;
  name: string;
  description: string;
  owner_id: number;
  agile_mode: 'scrum' | 'kanban';
  archived: boolean;
  created_at: string;
  updated_at: string;
}> = [];

export function resetFixtures() {
  currentUser = null;
  nextProjectId = 1;
  projects.length = 0;
}

export const handlers = [
  // ---------- Auth ----------
  http.post('http://localhost:8080/api/auth/register', async ({ request }) => {
    const body = (await request.json()) as Record<string, unknown>;
    const user = {
      id: 1,
      email: String(body.email || 'test@example.com'),
      username: String(body.email || 'test').split('@')[0],
      role: String(body.role || 'developer'),
      avatar_url: null,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    };
    const token = 'mock-jwt-token-' + Date.now();
    currentUser = { id: user.id, email: user.email, role: user.role, username: user.username };
    return HttpResponse.json({
      code: 0,
      message: '注册成功',
      data: { user, token, refresh_token: 'mock-refresh-' + Date.now(), expires_at: new Date(Date.now() + 86400000).toISOString() },
    });
  }),

  http.post('http://localhost:8080/api/auth/login', async ({ request }) => {
    const body = (await request.json()) as Record<string, unknown>;
    const token = 'mock-jwt-token-' + Date.now();
    const user = {
      id: 1,
      email: String(body.email || 'test@example.com'),
      username: 'testuser',
      role: 'developer',
      avatar_url: null,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    };
    currentUser = { id: user.id, email: user.email, role: user.role, username: user.username };
    return HttpResponse.json({
      code: 0,
      message: '登录成功',
      data: { user, token, refresh_token: 'mock-refresh-' + Date.now(), expires_at: new Date(Date.now() + 86400000).toISOString() },
    });
  }),

  http.post('http://localhost:8080/api/auth/refresh', async () => {
    return HttpResponse.json({
      code: 0,
      message: '刷新成功',
      data: { token: 'mock-refreshed-token-' + Date.now(), expires_at: new Date(Date.now() + 86400000).toISOString() },
    });
  }),

  // ---------- Projects ----------
  http.post('http://localhost:8080/api/projects', async ({ request }) => {
    const body = (await request.json()) as Record<string, unknown>;
    const project = {
      id: nextProjectId++,
      name: String(body.name || '未命名项目'),
      description: String(body.description || ''),
      owner_id: currentUser?.id ?? 1,
      agile_mode: (body.agile_mode as 'scrum' | 'kanban') || 'kanban',
      archived: false,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    };
    projects.push(project);
    return HttpResponse.json({
      code: 0,
      message: '项目创建成功',
      data: project,
    });
  }),

  http.get('http://localhost:8080/api/projects', async () => {
    return HttpResponse.json({
      code: 0,
      message: 'success',
      data: {
        projects,
        total: projects.length,
        page: 1,
        limit: 20,
      },
    });
  }),

  http.get('http://localhost:8080/api/projects/:id', async ({ params }) => {
    const id = Number(params.id);
    const project = projects.find((p) => p.id === id);
    if (!project) {
      return HttpResponse.json({ code: 1, message: '项目不存在', data: null }, { status: 404 });
    }
    return HttpResponse.json({ code: 0, message: 'success', data: project });
  }),

  // ---------- Stories ----------
  http.post('http://localhost:8080/api/projects/:id/stories', async ({ request, params }) => {
    const body = (await request.json()) as Record<string, unknown>;
    const story = {
      id: Date.now(),
      project_id: Number(params.id),
      title: String(body.title || '未命名故事'),
      description: String(body.description || ''),
      story_type: (body.story_type as string) || 'feature',
      status: 'backlog',
      priority: 0,
      points: (body.story_points as number) || 1,
      created_by: currentUser?.id ?? 1,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
      acceptance_criteria: [],
      tags: [],
    };
    return HttpResponse.json({ code: 0, message: '用户故事创建成功', data: story });
  }),

  http.get('http://localhost:8080/api/projects/:id/stories', async () => {
    return HttpResponse.json({
      code: 0,
      message: 'success',
      data: [],
    });
  }),

  // ---------- Me ----------
  http.get('http://localhost:8080/api/me/dashboard', async () => {
    return HttpResponse.json({
      code: 0,
      message: 'success',
      data: { projects: [], recent_stories: [], notifications: { unread_count: 0 } },
    });
  }),

  // ---------- Notifications ----------
  http.get('http://localhost:8080/api/notifications/unread-count', async () => {
    return HttpResponse.json({ code: 0, message: 'success', data: { count: 0 } });
  }),

  http.get('http://localhost:8080/api/notifications', async () => {
    return HttpResponse.json({
      code: 0,
      message: 'success',
      data: { items: [], total: 0, page: 1, limit: 20 },
    });
  }),

  // ---------- Search ----------
  http.get('http://localhost:8080/api/search/capabilities', async () => {
    return HttpResponse.json({
      code: 0,
      message: 'success',
      data: { semantic_search: false, filters: {} },
    });
  }),
];
