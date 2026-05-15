import { motion } from 'framer-motion';
import { FileSpreadsheet, Activity, BarChart3, PieChart } from 'lucide-react';

export function StatsGrid({ data }) {
  const cards = [
    { label: 'Total Files', value: data?.total_files || 0, icon: FileSpreadsheet, color: 'emerald' },
    { label: 'Total Size', value: formatBytes(data?.total_size || 0), icon: Activity, color: 'blue' },
    { label: 'Avg Size', value: formatBytes(data?.avg_size || 0), icon: BarChart3, color: 'indigo' },
    { label: 'Unique Types', value: data?.unique_types || 0, icon: PieChart, color: 'purple' },
  ];

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
      {cards.map((card, index) => (
        <motion.div
          key={card.label}
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: index * 0.1, type: "spring", stiffness: 100 }}
          className="bg-white/80 backdrop-blur-xl rounded-2xl p-6 shadow-sm border border-white/50 hover:shadow-lg transition-all relative overflow-hidden group"
        >
          <div className={`absolute top-0 right-0 w-32 h-32 bg-${card.color}-500/5 rounded-full blur-3xl -mr-16 -mt-16 transition-all group-hover:scale-150`} />
          <div className="flex items-center justify-between mb-4 relative z-10">
            <span className="text-sm font-medium text-gray-500 tracking-wide uppercase">{card.label}</span>
            <div className={`w-10 h-10 rounded-xl bg-${card.color}-50 flex items-center justify-center text-${card.color}-500`}>
              <card.icon className="w-5 h-5" />
            </div>
          </div>
          <p className="text-3xl font-bold text-gray-900 relative z-10">{card.value}</p>
        </motion.div>
      ))}
    </div>
  );
}

function formatBytes(bytes) {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
}
