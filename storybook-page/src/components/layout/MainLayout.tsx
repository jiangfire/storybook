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
        <div className="mx-auto mt-1 flex w-full max-w-[1600px] items-start gap-2 px-2 pb-6 sm:px-3 lg:mt-1.5 lg:gap-2.5 lg:px-4">
          {showSidebar && <Sidebar currentProject={currentProject} />}
          <main className="min-w-0 flex-1">
            <Outlet />
          </main>
        </div>
      </div>
    </div>
  );
}
