import { useForm } from 'react-hook-form';
import { useNavigate, Link } from 'react-router-dom';
import { authApi } from '../api/auth';
import { Button } from '../components/ui/Button';
import { Input } from '../components/ui/Input';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '../components/ui/Card';
import styles from './Auth.module.css';
import { useMutation } from '@tanstack/react-query';

export function Register() {
  const navigate = useNavigate();
  const { register, handleSubmit, formState: { errors } } = useForm();
  
  const registerMutation = useMutation({
    mutationFn: (data: any) => authApi.register(data.email, data.password),
    onSuccess: () => {
      navigate('/login');
    }
  });

  const onSubmit = (data: any) => {
    registerMutation.mutate(data);
  };

  return (
    <div className={styles.container}>
      <Card className={styles.card}>
        <CardHeader>
          <CardTitle>Create Account</CardTitle>
          <CardDescription>Join the platform to run distributed jobs</CardDescription>
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
              {...register('password', { required: 'Password is required', minLength: { value: 6, message: 'Minimum 6 characters' } })}
              error={errors.password?.message as string}
            />
            
            {registerMutation.isError && (
              <p className="text-danger text-sm">Failed to create account. Email might be in use.</p>
            )}

            <Button type="submit" isLoading={registerMutation.isPending} className="mt-4">
              Register
            </Button>
          </form>
          <div className={styles.link}>
            Already have an account? <Link to="/login">Sign In</Link>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
