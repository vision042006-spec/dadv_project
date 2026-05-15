import { useState, createContext, useContext, useEffect } from 'react';
import { motion } from 'framer-motion';
import { uploadFile, getJobStatus, login, signup, getMe } from '../lib/api';

const AuthContext = createContext(null);

export function AuthProvider({ children }) {
  const [user, setUser] = useState(null);
  const [loading, setLoading] = useState(true);
  const [token, setToken] = useState(localStorage.getItem('token'));
  const [refreshToken, setRefreshToken] = useState(localStorage.getItem('refresh_token'));

  useEffect(() => {
    checkAuth();
  }, []);

  const checkAuth = async () => {
    if (!token) {
      setLoading(false);
      return;
    }
    try {
      const userData = await getMe(token);
      setUser(userData);
    } catch (err) {
      logout();
    } finally {
      setLoading(false);
    }
  };

  const login = async (email, password) => {
    const response = await fetch('http://localhost:8080/api/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, password }),
    });
    const data = await response.json();
    if (!response.ok) throw new Error(data.error);
    
    localStorage.setItem('token', data.token);
    localStorage.setItem('refresh_token', data.refresh_token);
    setToken(data.token);
    setRefreshToken(data.refresh_token);
    setUser(data.user);
    return data;
  };

  const signup = async (email, password, name) => {
    const response = await fetch('http://localhost:8080/api/auth/signup', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, password, name }),
    });
    const data = await response.json();
    if (!response.ok) throw new Error(data.error);
    
    localStorage.setItem('token', data.token);
    localStorage.setItem('refresh_token', data.refresh_token);
    setToken(data.token);
    setRefreshToken(data.refresh_token);
    setUser(data.user);
    return data;
  };

  const googleLogin = async (googleData) => {
    const response = await fetch('http://localhost:8080/api/auth/google', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(googleData),
    });
    const data = await response.json();
    if (!response.ok) throw new Error(data.error);
    
    localStorage.setItem('token', data.token);
    localStorage.setItem('refresh_token', data.refresh_token);
    setToken(data.token);
    setRefreshToken(data.refresh_token);
    setUser(data.user);
    return data;
  };

  const logout = () => {
    localStorage.removeItem('token');
    localStorage.removeItem('refresh_token');
    setToken(null);
    setRefreshToken(null);
    setUser(null);
  };

  return (
    <AuthContext.Provider value={{ user, loading, token, login, signup, googleLogin, logout }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  return useContext(AuthContext);
}