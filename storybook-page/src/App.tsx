import { ToastProvider } from './components/ui/Toast';
import ErrorBoundary from './components/ErrorBoundary';
import AppRouter from './router';

const App = () => {
  return (
    <ErrorBoundary>
      <ToastProvider>
        <AppRouter />
      </ToastProvider>
    </ErrorBoundary>
  );
};

export default App;
