import { lazy, Suspense } from 'react';
import { createBrowserRouter, RouterProvider, Navigate } from 'react-router-dom';
import ProtectedRoute from '../components/ProtectedRoute';
import MainLayout from '../components/layout/MainLayout';
import LoginPage from '../pages/Auth/LoginPage';
import RegisterPage from '../pages/Auth/RegisterPage';
import LoadingSpinner from '../components/ui/LoadingSpinner';

// 懒加载页面组件
const ProjectListPage = lazy(() => import('../pages/Projects/ProjectListPage'));
const ProjectDetailPage = lazy(() => import('../pages/Projects/ProjectDetailPage'));
const BoardViewPage = lazy(() => import('../pages/Projects/BoardViewPage'));
const StoryDetailPage = lazy(() => import('../pages/Stories/StoryDetailPage'));
const StoryCreatePage = lazy(() => import('../pages/Stories/StoryCreatePage'));
const DashboardPage = lazy(() => import('../pages/Dashboard/DashboardPage'));
const TechLeadReviewPage = lazy(() => import('../pages/TechLead/ReviewPage'));
const TechLeadWorkloadPage = lazy(() => import('../pages/TechLead/WorkloadPage'));
const UserManagementPage = lazy(() => import('../pages/Admin/UserManagementPage'));

const router = createBrowserRouter([
  {
    path: '/login',
    element: <LoginPage />,
  },
  {
    path: '/register',
    element: <RegisterPage />,
  },
  {
    path: '/',
    element: (
      <ProtectedRoute>
        <MainLayout />
      </ProtectedRoute>
    ),
    children: [
      {
        index: true,
        element: <Navigate to="/dashboard" replace />,
      },
      {
        path: 'dashboard',
        element: (
          <Suspense fallback={<LoadingSpinner />}>
            <DashboardPage />
          </Suspense>
        ),
      },
      {
        path: 'projects',
        element: (
          <Suspense fallback={<LoadingSpinner />}>
            <ProjectListPage />
          </Suspense>
        ),
      },
      {
        path: 'projects/:id',
        element: (
          <Suspense fallback={<LoadingSpinner />}>
            <ProjectDetailPage />
          </Suspense>
        ),
      },
      {
        path: 'projects/:id/board',
        element: (
          <Suspense fallback={<LoadingSpinner />}>
            <BoardViewPage />
          </Suspense>
        ),
      },
      {
        path: 'stories/:id',
        element: (
          <Suspense fallback={<LoadingSpinner />}>
            <StoryDetailPage />
          </Suspense>
        ),
      },
      {
        path: 'projects/:projectId/stories/new',
        element: (
          <Suspense fallback={<LoadingSpinner />}>
            <StoryCreatePage />
          </Suspense>
        ),
      },
      {
        path: 'techlead/review',
        element: (
          <Suspense fallback={<LoadingSpinner />}>
            <TechLeadReviewPage />
          </Suspense>
        ),
      },
      {
        path: 'techlead/workload',
        element: (
          <Suspense fallback={<LoadingSpinner />}>
            <TechLeadWorkloadPage />
          </Suspense>
        ),
      },
      {
        path: 'admin/users',
        element: (
          <Suspense fallback={<LoadingSpinner />}>
            <UserManagementPage />
          </Suspense>
        ),
      },
    ],
  },
  {
    path: '*',
    element: <Navigate to="/projects" replace />,
  },
]);

export default function AppRouter() {
  return <RouterProvider router={router} />;
}
