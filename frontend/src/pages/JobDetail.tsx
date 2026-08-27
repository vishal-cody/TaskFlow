import { useParams, Link } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { jobsApi } from '../api/jobs';
import { Card, CardHeader, CardTitle, CardContent } from '../components/ui/Card';
import { Button } from '../components/ui/Button';
import { ArrowLeft, XCircle } from 'lucide-react';
import styles from './Jobs.module.css';

export function JobDetail() {
  const { id } = useParams<{ id: string }>();
  const queryClient = useQueryClient();

  const { data: job, isLoading } = useQuery({
    queryKey: ['job', id],
    queryFn: () => jobsApi.get(id!),
    refetchInterval: (query) => {
      // stop polling if terminal status
      const state = query.state.data?.status;
      if (state === 'completed' || state === 'failed' || state === 'cancelled') {
        return false;
      }
      return 2000; // poll every 2s
    },
    enabled: !!id
  });

  const { data: logs } = useQuery({
    queryKey: ['job-logs', id],
    queryFn: () => jobsApi.logs(id!),
    refetchInterval: (_query) => {
      const state = job?.status;
      if (state === 'completed' || state === 'failed' || state === 'cancelled') {
        return false;
      }
      return 2000;
    },
    enabled: !!id
  });

  const cancelMutation = useMutation({
    mutationFn: () => jobsApi.cancel(id!),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['job', id] });
      queryClient.invalidateQueries({ queryKey: ['job-logs', id] });
    }
  });

  if (isLoading) return (
    <div className="flex justify-center items-center h-full min-h-[50vh]">
      <p className="text-secondary">Loading job details...</p>
    </div>
  );
  if (!job) return (
    <div className="flex flex-col justify-center items-center h-full min-h-[50vh] gap-4">
      <h2 className="text-danger">Job Not Found</h2>
      <p className="text-secondary">This job may have been deleted or never existed.</p>
      <Link to="/jobs">
        <Button variant="secondary">Return to Jobs</Button>
      </Link>
    </div>
  );

  const isTerminal = job.status === 'completed' || job.status === 'failed' || job.status === 'cancelled';

  return (
    <div className="w-full">
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-4">
          <Link to="/jobs" className="text-secondary hover:text-primary">
            <ArrowLeft size={24} />
          </Link>
          <h2>Job Details</h2>
        </div>
        {!isTerminal && (
          <Button 
            variant="danger" 
            onClick={() => cancelMutation.mutate()}
            isLoading={cancelMutation.isPending}
          >
            <XCircle size={16} style={{ marginRight: '8px' }} /> Cancel Job
          </Button>
        )}
      </div>

      <div className={styles.detailGrid}>
        <div className="flex-col gap-6" style={{ display: 'flex' }}>
          <Card>
            <CardHeader>
              <CardTitle>Overview</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <p className="text-sm text-secondary font-medium">Job ID</p>
                  <p className="font-mono">{job.id}</p>
                </div>
                <div>
                  <p className="text-sm text-secondary font-medium">Type</p>
                  <p>{job.type}</p>
                </div>
                <div>
                  <p className="text-sm text-secondary font-medium">Status</p>
                  <span className={`${styles.badge} ${styles[job.status]}`}>{job.status}</span>
                </div>
                <div>
                  <p className="text-sm text-secondary font-medium">Priority</p>
                  <p>{job.priority}</p>
                </div>
                <div>
                  <p className="text-sm text-secondary font-medium">Created At</p>
                  <p>{new Date(job.created_at).toLocaleString()}</p>
                </div>
                {job.started_at && (
                  <div>
                    <p className="text-sm text-secondary font-medium">Started At</p>
                    <p>{new Date(job.started_at).toLocaleString()}</p>
                  </div>
                )}
                {job.completed_at && (
                  <div>
                    <p className="text-sm text-secondary font-medium">Finished At</p>
                    <p>{new Date(job.completed_at).toLocaleString()}</p>
                  </div>
                )}
                <div>
                  <p className="text-sm text-secondary font-medium">Retries</p>
                  <p>{job.retry_count} / {job.max_retries}</p>
                </div>
              </div>

              <div className="mt-6">
                <p className="text-sm text-secondary font-medium mb-2">Progress ({job.progress}%)</p>
                <div className={styles.progressBar}>
                  <div className={styles.progressFill} style={{ width: `${job.progress}%` }} />
                </div>
              </div>
            </CardContent>
          </Card>

          {job.result && (
            <Card>
              <CardHeader>
                <CardTitle>Result Data</CardTitle>
              </CardHeader>
              <CardContent>
                <pre className="text-sm font-mono bg-base p-4 rounded-md overflow-x-auto" style={{ backgroundColor: '#000', padding: '1rem', borderRadius: '0.375rem' }}>
                  {JSON.stringify(job.result, null, 2)}
                </pre>
              </CardContent>
            </Card>
          )}

          {job.error && (
            <Card>
              <CardHeader>
                <CardTitle className="text-danger">Error Details</CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-danger">{job.error}</p>
              </CardContent>
            </Card>
          )}
        </div>

        <div>
          <Card className="h-full">
            <CardHeader>
              <CardTitle>Execution Logs</CardTitle>
            </CardHeader>
            <CardContent>
              <div className={styles.logContainer}>
                {logs?.length === 0 ? (
                  <p className="text-muted">No logs emitted yet...</p>
                ) : (
                  (logs || []).map((log) => (
                    <div key={log.id} className={styles.logLine}>
                      <span className={styles.logTime}>
                        {new Date(log.timestamp).toLocaleTimeString()}
                      </span>
                      <span className={`${styles.logLevel} ${styles[log.level]}`}>
                        [{log.level}]
                      </span>
                      <span className={styles.logMsg}>{log.message}</span>
                    </div>
                  ))
                )}
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}
