import { motion } from 'framer-motion';
import { Upload, FileSpreadsheet, Loader2, Shield } from 'lucide-react';
import { Link } from 'react-router-dom';

export function Sidebar({ jobs, currentJob, onSelectJob, onUpload, loading }) {
  const handleDrop = (e) => {
    e.preventDefault();
    const file = e.dataTransfer.files[0];
    if (file) onUpload(file);
  };

  return (
    <aside className="w-full md:w-72 bg-white/80 backdrop-blur-xl border-b md:border-r border-gray-200/50 p-6 shadow-sm">
      <div className="mb-8">
        <h2 className="text-sm font-semibold text-gray-800 mb-4 tracking-wide uppercase">Upload Dataset</h2>
        <motion.div
          className="border-2 border-dashed border-gray-200 rounded-2xl p-6 text-center bg-gray-50/50 hover:border-emerald-400 hover:bg-emerald-50/30 transition-all cursor-pointer group"
          onDrop={handleDrop}
          onDragOver={(e) => e.preventDefault()}
          onClick={() => document.getElementById('file-input').click()}
          whileHover={{ scale: 1.02 }}
          whileTap={{ scale: 0.98 }}
        >
          {loading ? (
            <motion.div animate={{ rotate: 360 }} transition={{ duration: 1, repeat: Infinity, ease: "linear" }}>
              <Loader2 className="w-8 h-8 text-emerald-500 mx-auto" />
            </motion.div>
          ) : (
            <>
              <div className="w-12 h-12 bg-white rounded-full flex items-center justify-center mx-auto mb-3 shadow-sm group-hover:shadow-md transition-shadow">
                <Upload className="w-6 h-6 text-emerald-500" />
              </div>
              <p className="text-sm font-medium text-gray-700">Drop file here</p>
              <p className="text-xs text-gray-400 mt-1">CSV or JSON up to 100MB</p>
            </>
          )}
          <input
            id="file-input"
            type="file"
            accept=".csv,.json,.xlsx,.xls"
            className="hidden"
            onChange={(e) => e.target.files[0] && onUpload(e.target.files[0])}
          />
        </motion.div>
      </div>

      <div>
        <h2 className="text-sm font-semibold text-gray-800 mb-4 tracking-wide uppercase">Recent Jobs</h2>
        {jobs.length === 0 ? (
          <div className="text-center py-8">
            <div className="w-12 h-12 bg-gray-100 rounded-full flex items-center justify-center mx-auto mb-3">
              <FileSpreadsheet className="w-6 h-6 text-gray-300" />
            </div>
            <p className="text-sm text-gray-500">No files yet</p>
          </div>
        ) : (
          <div className="space-y-2">
            {jobs.map((jobId) => (
              <motion.button
                key={jobId}
                onClick={() => onSelectJob(jobId)}
                className={`w-full text-left px-4 py-3 rounded-xl text-sm transition-all flex items-center gap-3 ${
                  currentJob === jobId
                    ? 'bg-emerald-500 text-white shadow-md shadow-emerald-500/20'
                    : 'bg-white text-gray-600 border border-gray-100 hover:border-emerald-200 hover:bg-emerald-50/50'
                }`}
                whileHover={{ scale: 1.02 }}
                whileTap={{ scale: 0.98 }}
              >
                <div className={`w-8 h-8 rounded-lg flex items-center justify-center ${currentJob === jobId ? 'bg-white/20' : 'bg-gray-50'}`}>
                  <FileSpreadsheet className={`w-4 h-4 ${currentJob === jobId ? 'text-white' : 'text-gray-400'}`} />
                </div>
                <div className="flex-1 min-w-0">
                  <p className="font-medium truncate">Job {jobId.slice(0, 6)}</p>
                  <p className={`text-xs truncate ${currentJob === jobId ? 'text-emerald-100' : 'text-gray-400'}`}>{jobId}</p>
                </div>
              </motion.button>
            ))}
          </div>
        )}
      </div>

      <div className="mt-auto pt-6 border-t border-gray-100">
        <Link
          to="/dev-panel"
          className="w-full text-left px-4 py-3 rounded-xl text-sm transition-all flex items-center gap-3 bg-white text-gray-600 border border-gray-100 hover:border-emerald-200 hover:bg-emerald-50/50"
        >
          <div className="w-8 h-8 rounded-lg bg-gray-50 flex items-center justify-center">
            <Shield className="w-4 h-4 text-gray-400" />
          </div>
          <div className="flex-1 min-w-0">
            <p className="font-medium">Dev Panel</p>
            <p className="text-xs text-gray-400">System metrics</p>
          </div>
        </Link>
      </div>
    </aside>
  );
}
