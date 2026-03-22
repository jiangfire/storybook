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
        <div className="mx-auto mt-5 flex w-full max-w-[1600px] items-start gap-5 px-3 pb-8 sm:px-4 lg:mt-6 lg:px-6 xl:gap-7">
          {showSidebar && <Sidebar currentProject={currentProject} />}
          <main className="min-w-0 flex-1">
            <Outlet />
          </main>
        </div>
      </div>
    </div>
  );
}
