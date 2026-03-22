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
    <div className="app-shell">
      <div className="app-surface">
        <Header />
        <div className="mx-auto flex w-full max-w-[1600px] items-start gap-4 px-3 pb-8 sm:px-4 lg:px-6 xl:gap-6">
          {showSidebar && <Sidebar currentProject={currentProject} />}
          <main className="min-w-0 flex-1 pt-2 md:pt-6">
            <Outlet />
          </main>
        </div>
      </div>
    </div>
  );
}
