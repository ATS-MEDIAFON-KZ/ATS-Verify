import { useState, type FormEvent } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { useAuth } from '../hooks/useAuth';
import api from '../lib/api';
import { Lock, User, CheckCircle2 } from 'lucide-react';
import toast from 'react-hot-toast';

export default function RegisterPage() {
    const [username, setUsername] = useState('');
    const [password, setPassword] = useState('');
    const [error, setError] = useState('');
    const [loading, setLoading] = useState(false);
    const { login } = useAuth();
    const navigate = useNavigate();

    const handleSubmit = async (e: FormEvent) => {
        e.preventDefault();
        setError('');
        setLoading(true);

        // Client-side validation
        if (password.length < 6) {
            setError('Пароль должен содержать минимум 6 символов');
            setLoading(false);
            return;
        }

        try {
            const { data } = await api.post('/auth/register', { username, password });

            toast.success(data.message || 'Регистрация успешна');

            if (data.user.is_approved) {
                // If auto-approved (e.g. ATS domain), log them in immediately
                const loginRes = await api.post('/auth/login', { username, password });
                const loginData = loginRes.data;
                login({
                    id: loginData.user.id,
                    username: loginData.user.username,
                    role: loginData.user.role,
                    marketplace_prefix: loginData.user.marketplace_prefix,
                    token: loginData.token
                });
                navigate('/');
            } else {
                // Return to login to wait for approval
                setTimeout(() => {
                    navigate('/login');
                }, 2000);
            }
        } catch (err: unknown) {
            const errData = (err as any).response?.data;
            const msg = errData?.message || errData?.error;
            setError(msg || 'Ошибка регистрации');
        } finally {
            setLoading(false);
        }
    };

    return (
        <div className="min-h-screen flex items-center justify-center bg-bg px-4">
            <div className="w-full max-w-md">
                {/* Logo */}
                <div className="text-center mb-8">
                    <div className="inline-flex items-center justify-center w-14 h-14 rounded-2xl bg-primary mb-4">
                        <CheckCircle2 size={28} className="text-white" />
                    </div>
                    <h1 className="text-3xl font-bold text-text-primary">ATS-Verify</h1>
                    <p className="text-text-muted mt-1 text-sm">Платформа верификации</p>
                </div>

                {/* Register Card */}
                <div className="card p-8">
                    <h2 className="text-lg font-semibold text-text-primary mb-6">Регистрация</h2>

                    <form onSubmit={handleSubmit} className="space-y-4">
                        {error && (
                            <div className="bg-danger-light border border-danger/20 text-danger text-sm px-4 py-3 rounded-lg">
                                {error}
                            </div>
                        )}

                        <div>
                            <label className="text-sm font-medium text-text-primary block mb-1.5">Логин (Email)</label>
                            <div className="relative">
                                <User size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-text-muted" />
                                <input
                                    type="text"
                                    value={username}
                                    onChange={(e) => setUsername(e.target.value)}
                                    className="input"
                                    style={{ paddingLeft: '2.5rem' }}
                                    placeholder="Введите email"
                                    required
                                />
                            </div>
                        </div>

                        <div>
                            <label className="text-sm font-medium text-text-primary block mb-1.5">Пароль</label>
                            <div className="relative">
                                <Lock size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-text-muted" />
                                <input
                                    type="password"
                                    value={password}
                                    onChange={(e) => setPassword(e.target.value)}
                                    className="input"
                                    style={{ paddingLeft: '2.5rem' }}
                                    placeholder="Минимум 6 символов"
                                    required
                                    minLength={6}
                                />
                            </div>
                        </div>

                        <button type="submit" disabled={loading} className="btn-primary w-full justify-center py-2.5">
                            {loading ? 'Создание...' : 'Зарегистрироваться'}
                        </button>
                    </form>

                    <div className="mt-6 text-center">
                        <p className="text-sm text-text-muted">
                            Уже есть аккаунт?{' '}
                            <Link to="/login" className="font-semibold text-primary hover:text-primary-hover">
                                Войти в систему
                            </Link>
                        </p>
                    </div>
                </div>

                <p className="text-center text-text-muted text-xs mt-6">
                    © 2026 ATS-Verify. Все права защищены.
                </p>
            </div>
        </div>
    );
}
