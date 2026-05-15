import { useState, useEffect } from 'react';
import { Routes, Route, Navigate } from 'react-router-dom';
import { motion, AnimatePresence } from 'framer-motion';
import { BarChart3, Loader2, LogOut } from 'lucide-react';
import { uploadFile, getJobStatus, getAggregateStats, getFileTypeStats, getSizeDistribution, getAnomalies } from './lib/api';
import { useAuth } from './context/AuthContext';
import Login from './pages/Login';
import Signup from './pages/Signup';

import { Sidebar } from './components/Sidebar';
import { StatsGrid } from './components/StatsGrid';
import { AnalysisCharts } from './components/AnalysisCharts';
import { AnomalyList } from './components/AnomalyList';

function ProtectedRoute({ children }) {
  const { user, loading } = useAuth();
  
  if (loading) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center">
        <Loader2 className="w-8 h-8 text-emerald-500 animate-spin" />
      </div>
    );
  }
  
  return user ? children : <Navigate to="/login" />;
}

function App() {
  return (
    <Routes>
      <Route path="/login" element={<Login />} />
      <Route path="/signup" element={<Signup />} />
      <Route path="/*" element={
        <ProtectedRoute>
          <Dashboard />
        </ProtectedRoute>
      } />
    </Routes>
  );
}

function Dashboard() {
  const [currentJob, setCurrentJob] = useState(null);
  const [jobs, setJobs] = useState([]);
  const [loading, setLoading] = useState(false);
  const { user, logout } = useAuth();

  const handleUpload = async (file) => {
    setLoading(true);
    try {
      const result = await uploadFile(file);
      setCurrentJob(result.job_id);
      if (!jobs.includes(result.job_id)) {
        setJobs(prev => [...prev, result.job_id]);
      }
    } catch (err) {
      alert(err.message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-[#f8fafc] font-sans selection:bg-emerald-100 selection:text-emerald-900">
      {/* Header */}
      <header className="bg-white/80 backdrop-blur-xl border-b border-gray-200/50 px-6 py-4 sticky top-0 z-50">
        <div className="flex items-center justify-between max-w-[1600px] mx-auto">
          <div className="flex items-center gap-4">
            <div className="w-10 h-10 bg-gradient-to-br from-emerald-400 to-emerald-600 rounded-xl flex items-center justify-center shadow-lg shadow-emerald-500/20">
              <BarChart3 className="w-6 h-6 text-white" />
            </div>
            <div>
              <h1 className="text-xl font-bold text-gray-900 tracking-tight">DADV</h1>
              <p className="text-[11px] font-medium text-gray-500 uppercase tracking-wider">Metadata Intelligence</p>
            </div>
          </div>
          <div className="flex items-center gap-6">
            <div className="flex items-center gap-3">
              <div className="w-8 h-8 rounded-full bg-emerald-100 flex items-center justify-center text-emerald-700 font-bold text-sm">
                {user?.name?.[0]?.toUpperCase() || user?.email?.[0]?.toUpperCase()}
              </div>
              <div className="hidden md:block">
                <p className="text-sm font-semibold text-gray-700 leading-none">{user?.name || 'User'}</p>
                <p className="text-xs text-gray-400 mt-1">{user?.email}</p>
              </div>
            </div>
            <div className="h-6 w-px bg-gray-200"></div>
            <button
              onClick={logout}
              className="flex items-center gap-2 text-sm font-medium text-gray-500 hover:text-red-500 transition-colors"
            >
              <LogOut className="w-4 h-4" />
              <span>Sign out</span>
            </button>
          </div>
        </div>
      </header>

      <div className="flex max-w-[1600px] mx-auto">
        <Sidebar 
          jobs={jobs} 
          currentJob={currentJob} 
          onSelectJob={setCurrentJob} 
          onUpload={handleUpload}
          loading={loading}
        />

        <main className="flex-1 p-8 overflow-x-hidden">
          <AnimatePresence mode="wait">
            {currentJob ? (
              <motion.div
                key={currentJob}
                initial={{ opacity: 0, y: 20 }}
                animate={{ opacity: 1, y: 0 }}
                exit={{ opacity: 0, y: -20 }}
                transition={{ duration: 0.4, ease: "easeOut" }}
              >
                <DashboardContent jobId={currentJob} />
              </motion.div>
            ) : (
              <motion.div
                key="empty"
                initial={{ opacity: 0, scale: 0.95 }}
                animate={{ opacity: 1, scale: 1 }}
                exit={{ opacity: 0 }}
                className="h-[calc(100vh-140px)] flex items-center justify-center"
              >
                <EmptyState />
              </motion.div>
            )}
          </AnimatePresence>
        </main>
      </div>
    </div>
  );
}

function EmptyState() {
  return (
    <div className="text-center max-w-md mx-auto">
      <motion.div 
        animate={{ y: [0, -15, 0] }} 
        transition={{ duration: 4, repeat: Infinity, ease: "easeInOut" }}
        className="relative mb-8"
      >
        <div className="absolute inset-0 bg-emerald-500/10 blur-3xl rounded-full" />
        <BarChart3 className="w-24 h-24 text-gray-200 mx-auto relative z-10" />
      </motion.div>
      <h2 className="text-2xl font-bold text-gray-900 mb-3 tracking-tight">No Dataset Selected</h2>
      <p className="text-gray-500 leading-relaxed">
        Upload a file using the dropzone on the left to start analyzing your metadata and generating insights.
      </p>
    </div>
  );
}

function DashboardContent({ jobId }) {
  const [status, setStatus] = useState(null);
  const [data, setData] = useState({});

  useEffect(() => {
    let interval;
    const checkStatus = async () => {
      try {
        const s = await getJobStatus(jobId);
        setStatus(s);
        if (s.status === 'completed') {
          loadData();
          if (interval) clearInterval(interval);
        } else if (s.status === 'failed') {
          if (interval) clearInterval(interval);
        }
      } catch (err) {
        console.error(err);
      }
    };

    checkStatus();
    interval = setInterval(checkStatus, 2000);
    return () => clearInterval(interval);
  }, [jobId]);

  const loadData = async () => {
    try {
      const [agg, fileTypes, sizeDist, anomalies] = await Promise.all([
        getAggregateStats(jobId),
        getFileTypeStats(jobId),
        getSizeDistribution(jobId),
        getAnomalies(jobId),
      ]);
      setData({ aggregate: agg.data, fileTypes: fileTypes.data, sizeDist: sizeDist.data, anomalies: anomalies.data });
    } catch(e) {
      console.error(e);
    }
  };

  if (!status || status.status === 'pending' || status.status === 'processing') {
    return (
      <div className="h-[calc(100vh-140px)] flex flex-col items-center justify-center">
        <motion.div 
          animate={{ rotate: 360 }} 
          transition={{ duration: 2, repeat: Infinity, ease: "linear" }}
          className="relative mb-6"
        >
          <div className="absolute inset-0 bg-emerald-400/20 blur-xl rounded-full" />
          <Loader2 className="w-16 h-16 text-emerald-500 relative z-10" />
        </motion.div>
        <h2 className="text-xl font-bold text-gray-900 mb-2">Analyzing Data</h2>
        <p className="text-gray-500">We're crunching the numbers for your dataset.</p>
        <div className="mt-8 w-64 h-1.5 bg-gray-100 rounded-full overflow-hidden">
          <motion.div 
            className="h-full bg-emerald-500"
            animate={{ x: ["-100%", "100%"] }}
            transition={{ duration: 1.5, repeat: Infinity, ease: "easeInOut" }}
          />
        </div>
      </div>
    );
  }

  if (status.status === 'failed') {
     return (
        <div className="h-[calc(100vh-140px)] flex flex-col items-center justify-center">
          <h2 className="text-xl font-bold text-red-600 mb-2">Processing Failed</h2>
          <p className="text-gray-500">{status.error}</p>
        </div>
     );
  }

  return (
    <div className="max-w-6xl">
      <div className="mb-8">
        <h2 className="text-2xl font-bold text-gray-900">Analysis Overview</h2>
        <p className="text-gray-500 mt-1">Insights and metrics extracted from your dataset.</p>
      </div>
      <StatsGrid data={data.aggregate} />
      <AnalysisCharts data={data} />
      <AnomalyList anomalies={data.anomalies} />
    </div>
  );
}

export default App;