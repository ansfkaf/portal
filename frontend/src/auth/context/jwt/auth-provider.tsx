// src\auth\context\jwt\auth-provider.tsx
import { useMemo, useReducer, useCallback, useEffect } from 'react';
import { AuthContext } from '../auth-context';
import { TOKEN_STORAGE_KEY } from './constant';
import { getStoredUser } from './utils';

// ----------------------------------------------------------------------

enum Types {
  INITIAL = 'INITIAL',
  LOGIN = 'LOGIN',
  REGISTER = 'REGISTER',
  LOGOUT = 'LOGOUT',
}

type Payload = {
  user: any;
};

type Action = {
  type: Types;
  payload?: Payload;
};

const initialState = {
  user: null,
  loading: true,
};

const reducer = (state: any, action: Action) => {
  switch (action.type) {
    case Types.INITIAL:
      return {
        loading: false,
        user: action.payload?.user,
      };
    case Types.LOGIN:
      return {
        ...state,
        user: action.payload?.user,
      };
    case Types.REGISTER:
      return {
        ...state,
        user: action.payload?.user,
      };
    case Types.LOGOUT:
      return {
        ...state,
        user: null,
      };
    default:
      return state;
  }
};

// ----------------------------------------------------------------------

type AuthProviderProps = {
  children: React.ReactNode;
};

export function AuthProvider({ children }: AuthProviderProps) {
  const [state, dispatch] = useReducer(reducer, initialState);

  const status = state.loading
    ? 'loading'
    : state.user
    ? 'authenticated'
    : 'unauthenticated';

  const initialize = useCallback(async () => {
    try {
      const token = localStorage.getItem(TOKEN_STORAGE_KEY);

      if (token) {
        const user = getStoredUser();
        dispatch({
          type: Types.INITIAL,
          payload: {
            user,
          },
        });
      } else {
        dispatch({
          type: Types.INITIAL,
          payload: {
            user: null,
          },
        });
      }
    } catch (error) {
      console.error(error);
      dispatch({
        type: Types.INITIAL,
        payload: {
          user: null,
        },
      });
    }
  }, []);

  useEffect(() => {
    initialize();
  }, [initialize]);

  // CHECK USER SESSION
  const checkUserSession = useCallback(async () => {
    try {
      const token = localStorage.getItem(TOKEN_STORAGE_KEY);

      if (token) {
        const user = getStoredUser();
        dispatch({
          type: Types.LOGIN,
          payload: {
            user,
          },
        });
      } else {
        dispatch({
          type: Types.LOGOUT,
        });
      }
    } catch (error) {
      console.error(error);
      dispatch({
        type: Types.LOGOUT,
      });
    }
  }, []);

  const memoizedValue = useMemo(
    () => ({
      user: state.user,
      loading: status === 'loading',
      authenticated: status === 'authenticated',
      unauthenticated: status === 'unauthenticated',
      checkUserSession,
    }),
    [state.user, status, checkUserSession]
  );

  return <AuthContext.Provider value={memoizedValue}>{children}</AuthContext.Provider>;
}