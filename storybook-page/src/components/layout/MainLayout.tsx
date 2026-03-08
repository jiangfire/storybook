import { Outlet } from 'react-router-dom';
import Header from './Header';
import Sidebar from './Sidebar';
import { useProjectStore } from '../../stores/projectStore';

interface MainLayoutProps {
  showSidebar?: boolean;
}

export default function MainLayout({ showSidebar = true }: MainLayoutProps) {
  const { currentProject } = useProjectStore();

  return (
    <div className="min-h-screen bg-bg-gray">
      <Header />

      <div className="flex">
        {/* 侧边栏 */}
        {showSidebar && <Sidebar currentProject={currentProject} />}

        {/* 主内容区 */}
        <main className="flex-1 overflow-auto">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
