import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react';

import {
  getCurrentUser,
  login as loginRequest,
  logout as logoutRequest,
  register as registerRequest,
  type CurrentUser,
  type LoginInput,
  type RegisterInput,
} from '../api/auth';

type AuthState =
  | { status: 'loading'; user: null; error: null }
  | { status: 'anonymous'; user: null; error: string | null }
  | { status: 'authenticated'; user: CurrentUser; error: null };

type AuthContextValue = AuthState & {
  login: (input: LoginInput) => Promise<void>;
  logout: () => Promise<void>;
  register: (input: RegisterInput) => Promise<void>;
  refresh: () => Promise<void>;
};

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [state, setState] = useState<AuthState>({
    status: 'loading',
    user: null,
    error: null,
  });

  const refresh = useCallback(async () => {
    try {
      const user = await getCurrentUser();
      setState({ status: 'authenticated', user, error: null });
    } catch {
      setState({ status: 'anonymous', user: null, error: null });
    }
  }, []);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const login = useCallback(async (input: LoginInput) => {
    try {
      const user = await loginRequest(input);
      setState({ status: 'authenticated', user, error: null });
    } catch (error) {
      setState({
        status: 'anonymous',
        user: null,
        error: error instanceof Error ? error.message : 'Login failed',
      });
      throw error;
    }
  }, []);

  const register = useCallback(async (input: RegisterInput) => {
    try {
      await registerRequest(input);
      const user = await loginRequest({ email: input.email, password: input.password });
      setState({ status: 'authenticated', user, error: null });
    } catch (error) {
      setState({
        status: 'anonymous',
        user: null,
        error: error instanceof Error ? error.message : 'Register failed',
      });
      throw error;
    }
  }, []);

  const logout = useCallback(async () => {
    await logoutRequest();
    setState({ status: 'anonymous', user: null, error: null });
  }, []);

  const value = useMemo<AuthContextValue>(
    () => ({ ...state, login, logout, register, refresh }),
    [state, login, logout, register, refresh],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const value = useContext(AuthContext);
  if (!value) {
    throw new Error('useAuth must be used within AuthProvider');
  }
  return value;
}
