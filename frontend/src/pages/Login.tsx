import { useForm as useHookForm } from 'react-hook-form';
import { useNavigate, Link } from 'react-router-dom';
import { authApi } from '../api/auth';
import { Button } from '../components/ui/Button';
import { Input } from '../components/ui/Input';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '../components/ui/Card';
import styles from './Auth.module.css';
import { useMutation } from '@tanstack/react-query';

export function Login() {
  const navigate = useNavigate();
  const { register, handleSubmit, formState: { errors } } = useHookForm();
  
  const loginMutation = useMutation({
    mutationFn: (data: any) => authApi.login(data.email, data.password),
    onSuccess: (data) => {
      localStorage.setItem('token', data.access_token);
      navigate('/');
    }
  });

  const onSubmit = (data: any) => {
    loginMutation.mutate(data);
  };

  return (
    <div className={styles.container}>
      <Card className={styles.card}>
        <CardHeader>
          <CardTitle>Welcome Back</CardTitle>
          <CardDescription>Sign in to manage your jobs</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit(onSubmit)} className={styles.form}>
            <Input 
              label="Email" 
              type="email" 
              {...register('email', { required: 'Email is required' })}
              error={errors.email?.message as string}
            />
            <Input 
              label="Password" 
              type="password" 
              {...register('password', { required: 'Password is required' })}
              error={errors.password?.message as string}
            />
            
            {loginMutation.isError && (
              <p className="text-danger text-sm">Invalid credentials</p>
            )}

            <Button type="submit" isLoading={loginMutation.isPending} className="mt-4">
              Sign In
            </Button>
          </form>
          <div className={styles.link}>
            Don't have an account? <Link to="/register">Register</Link>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
