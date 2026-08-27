import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { jobsApi } from '../api/jobs';
import { Card } from '../components/ui/Card';
import { Button } from '../components/ui/Button';
import { Play } from 'lucide-react';
import styles from './Dashboard.module.css';

export function Dashboard() {
  const queryClient = useQueryClient();
  const [isSubmitting, setIsSubmitting] = useState(false);

  const { data: stats, isLoading } = useQuery({
    queryKey: ['job-stats'],
    queryFn: jobsApi.stats,
    refetchInterval: 5000 // poll every 5s for dashboard updates
  });

  const createJobMutation = useMutation({
    mutationFn: (type: string) => jobsApi.create({ type, priority: 5, payload: {} }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['job-stats'] });
    },
    onSettled: () => setIsSubmitting(false)
  });

  const handleCreateJob = (type: string) => {
    setIsSubmitting(true);
    createJobMutation.mutate(type);
  };

  return (
    <div>
      <div className={styles.actions}>
        <div className="flex gap-4">
          <Button 
            onClick={() => handleCreateJob('report_generation')} 
            isLoading={isSubmitting}
            variant="secondary"
          >
            <Play size={16} className="mr-2" style={{ marginRight: '8px' }} />
            Run Report Job
          </Button>
          <Button 
            onClick={() => handleCreateJob('data_processing')} 
            isLoading={isSubmitting}
            variant="primary"
          >
            <Play size={16} className="mr-2" style={{ marginRight: '8px' }} />
            Run Data Job
          </Button>
        </div>
      </div>

      <div className={styles.grid}>
        <Card className={`${styles.statCard}`}>
          <span className={styles.statLabel}>Total Jobs</span>
          <span className={styles.statValue}>
            {isLoading ? '-' : (stats?.total || 0)}
          </span>
        </Card>
        <Card className={`${styles.statCard} ${styles.completed}`}>
          <span className={styles.statLabel}>Completed</span>
          <span className={styles.statValue}>
            {isLoading ? '-' : (stats?.completed || 0)}
          </span>
        </Card>
        <Card className={`${styles.statCard} ${styles.processing}`}>
          <span className={styles.statLabel}>Processing</span>
          <span className={styles.statValue}>
            {isLoading ? '-' : (stats?.processing || 0)}
          </span>
        </Card>
        <Card className={`${styles.statCard} ${styles.failed}`}>
          <span className={styles.statLabel}>Failed</span>
          <span className={styles.statValue}>
            {isLoading ? '-' : (stats?.failed || 0)}
          </span>
        </Card>
        <Card className={`${styles.statCard} ${styles.queued}`}>
          <span className={styles.statLabel}>Queued</span>
          <span className={styles.statValue}>
            {isLoading ? '-' : (stats?.queued || 0)}
          </span>
        </Card>
      </div>
    </div>
  );
}
