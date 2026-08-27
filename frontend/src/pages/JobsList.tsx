import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import { jobsApi } from '../api/jobs';
import { Button } from '../components/ui/Button';
import styles from './Jobs.module.css';
import { ChevronLeft, ChevronRight, ExternalLink } from 'lucide-react';

export function JobsList() {
  const [page, setPage] = useState(1);
  
  const { data, isLoading } = useQuery({
    queryKey: ['jobs', page],
    queryFn: () => jobsApi.list(page, 10),
    refetchInterval: 5000
  });

  if (isLoading) return <div>Loading jobs...</div>;

  return (
    <div className="w-full">
      <div className="flex items-center justify-between mb-6">
        <h2>All Jobs</h2>
      </div>

      <div className={styles.tableWrapper}>
        <table>
          <thead>
            <tr>
              <th>ID</th>
              <th>Type</th>
              <th>Status</th>
              <th>Progress</th>
              <th>Created</th>
              <th>Action</th>
            </tr>
          </thead>
          <tbody>
            {(data?.jobs || []).map((job) => (
              <tr key={job.id}>
                <td className="font-mono text-sm">{job.id.substring(0, 8)}...</td>
                <td>{job.type}</td>
                <td>
                  <span className={`${styles.badge} ${styles[job.status]}`}>
                    {job.status}
                  </span>
                </td>
                <td>
                  <div className="flex items-center gap-2">
                    <span className="text-sm w-8">{job.progress}%</span>
                    <div className={styles.progressBar} style={{ width: '100px', marginTop: 0 }}>
                      <div className={styles.progressFill} style={{ width: `${job.progress}%` }} />
                    </div>
                  </div>
                </td>
                <td className="text-sm text-secondary">
                  {new Date(job.created_at).toLocaleString()}
                </td>
                <td>
                  <Link to={`/jobs/${job.id}`}>
                    <Button variant="ghost" size="sm">
                      <ExternalLink size={16} />
                    </Button>
                  </Link>
                </td>
              </tr>
            ))}
            {(!data?.jobs || data.jobs.length === 0) && (
              <tr>
                <td colSpan={6} style={{ textAlign: 'center' }}>No jobs found</td>
              </tr>
            )}
          </tbody>
        </table>

        {data && data.total_pages > 1 && (
          <div className={styles.pagination}>
            <span className={styles.paginationText}>
              Showing page {data.page} of {data.total_pages} ({data.total} total)
            </span>
            <div className="flex gap-2">
              <Button 
                variant="secondary" 
                size="sm" 
                disabled={page === 1}
                onClick={() => setPage(p => p - 1)}
              >
                <ChevronLeft size={16} />
              </Button>
              <Button 
                variant="secondary" 
                size="sm" 
                disabled={page === data.total_pages}
                onClick={() => setPage(p => p + 1)}
              >
                <ChevronRight size={16} />
              </Button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
