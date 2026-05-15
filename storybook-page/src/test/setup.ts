import '@testing-library/jest-dom/vitest';
import { cleanup } from '@testing-library/react';
import { beforeAll, afterAll, afterEach } from 'vitest';
import { server } from './server';
import { resetFixtures } from './handlers';

beforeAll(() => {
  server.listen({ onUnhandledRequest: 'error' });
});

afterEach(() => {
  cleanup();
  server.resetHandlers();
  resetFixtures();
});

afterAll(() => {
  server.close();
});
