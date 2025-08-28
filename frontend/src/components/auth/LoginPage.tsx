import React, { useState, useEffect } from 'react';
import { motion } from 'framer-motion';
import {
  Zap, Mail, Lock, ArrowRight, Eye, EyeOff, Activity, Layers, Network,
  AlertCircle, Shield
} from 'lucide-react';
import {
  ReactFlow,
  useNodesState,
  useEdgesState,
  ReactFlowProvider,
  Handle,
  Position,
} from '@xyflow/react';
import type { Node, Edge } from '@xyflow/react';
import '@xyflow/react/dist/style.css';
import { useAuthStore } from '../../store/authStore';
import type { LoginRequest } from '../../store/authStore';
import { useAuthHealthCheck } from '../../hooks/useAuthHealthCheck';
import { CertificateHelper } from '../ui/CertificateHelper';

interface LoginPageProps {
  onSuccess: () => void;
  onSwitchToSignup: () => void;
  onTryDemo: () => void;
}

// Custom Node Components
const SystemSimLogoNode = () => (
  <div className="w-[400px] h-[500px] flex flex-col justify-center items-center p-6 bg-transparent">
    <Handle type="source" position={Position.Right} className="w-3 h-3 bg-app-tertiary border-2 border-app-primary" />

    {/* Logo with Enhanced Design */}
    <motion.div
      initial={{ opacity: 0, x: -50 }}
      animate={{ opacity: 1, x: 0 }}
      transition={{ duration: 0.8 }}
      className="mb-8"
    >
      <div className="relative">
        <div className="w-24 h-24 bg-app-secondary rounded-3xl flex items-center justify-center border border-app-primary shadow-2xl relative overflow-hidden">
          {/* Background Pattern */}
          <div className="absolute inset-0 bg-gradient-to-br from-app-tertiary/20 to-transparent" />

          {/* Main Icon */}
          <div className="relative z-10 flex items-center justify-center">
            <Zap className="w-12 h-12 text-app-primary" />
          </div>

          {/* Animated Elements */}
          <motion.div
            animate={{ rotate: 360 }}
            transition={{ duration: 8, repeat: Infinity, ease: "linear" }}
            className="absolute inset-3 border border-app-tertiary/30 rounded-2xl"
          />
        </div>

        {/* Floating Icons */}
        <motion.div
          animate={{ y: [-2, 2, -2] }}
          transition={{ duration: 3, repeat: Infinity, ease: "easeInOut" }}
          className="absolute -top-2 -right-2 w-8 h-8 bg-app-tertiary rounded-xl flex items-center justify-center"
        >
          <Activity className="w-4 h-4 text-app-primary" />
        </motion.div>

        <motion.div
          animate={{ y: [2, -2, 2] }}
          transition={{ duration: 3, repeat: Infinity, ease: "easeInOut", delay: 1 }}
          className="absolute -bottom-2 -left-2 w-8 h-8 bg-app-tertiary rounded-xl flex items-center justify-center"
        >
          <Network className="w-4 h-4 text-app-primary" />
        </motion.div>
      </div>
    </motion.div>

    {/* Title and Tagline */}
    <motion.div
      initial={{ opacity: 0, x: -50 }}
      animate={{ opacity: 1, x: 0 }}
      transition={{ duration: 0.8, delay: 0.2 }}
      className="text-center max-w-sm"
    >
      <h1 className="text-4xl font-bold text-app-primary mb-4">SystemSim</h1>
      <div className="flex items-center justify-center gap-3 mb-4">
        <div className="w-8 h-0.5 bg-app-tertiary rounded-full" />
        <Layers className="w-4 h-4 text-app-tertiary" />
        <div className="w-8 h-0.5 bg-app-tertiary rounded-full" />
      </div>
      <p className="text-lg text-app-secondary font-medium mb-2">
        Professional System Design Simulator
      </p>
      <p className="text-app-tertiary text-sm leading-relaxed mb-4">
        Build, test, and optimize distributed systems with realistic simulation.
      </p>

      {/* Feature highlights */}
      <div className="space-y-2">
        <div className="flex items-center justify-center gap-2 text-app-tertiary">
          <Activity className="w-3 h-3" />
          <span className="text-xs">Real-time monitoring</span>
        </div>
        <div className="flex items-center justify-center gap-2 text-app-tertiary">
          <Network className="w-3 h-3" />
          <span className="text-xs">System modeling</span>
        </div>
        <div className="flex items-center justify-center gap-2 text-app-tertiary">
          <Layers className="w-3 h-3" />
          <span className="text-xs">Component architecture</span>
        </div>
      </div>
    </motion.div>
  </div>
);

const AuthSystemNode = ({ data }: { data: any }) => {
  const { isLoginLoading, loginError, healthStatus } = data;

  return (
    <div className="w-[100px] h-[80px] flex flex-col items-center justify-center relative">
      <Handle type="target" position={Position.Left} className="w-3 h-3 bg-app-tertiary border-2 border-app-primary" />
      <Handle type="source" position={Position.Right} className="w-3 h-3 bg-app-tertiary border-2 border-app-primary" />

      {/* Simple Auth System Icon */}
      <motion.div
        animate={isLoginLoading || healthStatus === 'checking' ? {
          scale: [1, 1.1, 1]
        } : {}}
        transition={{ duration: 1.5, repeat: (isLoginLoading || healthStatus === 'checking') ? Infinity : 0 }}
        className="w-12 h-12 bg-app-secondary/80 backdrop-blur-xl rounded-xl border border-app-primary/30 flex items-center justify-center shadow-xl relative z-10"
      >
        <Shield className="w-6 h-6 text-app-tertiary" />

        {/* Status indicator */}
        <motion.div
          className={`absolute -top-1 -right-1 w-3 h-3 rounded-full border border-app-secondary ${
            loginError ? 'bg-red-400' :
            isLoginLoading ? 'bg-yellow-400' :
            healthStatus === 'checking' ? 'bg-blue-400' :
            healthStatus === 'healthy' ? 'bg-green-400' : 'bg-red-400'
          }`}
          animate={healthStatus === 'checking' ? { scale: [1, 1.2, 1] } : {}}
          transition={{ duration: 1, repeat: healthStatus === 'checking' ? Infinity : 0 }}
        />
      </motion.div>

      {/* Auth System Label with Health Status */}
      <motion.div
        className="mt-1 text-center relative z-10"
        animate={isLoginLoading ? { opacity: [1, 0.7, 1] } : {}}
        transition={{ duration: 2, repeat: isLoginLoading ? Infinity : 0 }}
      >
        <div className="text-xs font-bold text-app-secondary">Auth System</div>
        <div className={`text-xs font-medium ${
          loginError ? 'text-red-400' :
          isLoginLoading ? 'text-yellow-400' :
          healthStatus === 'checking' ? 'text-blue-400' :
          healthStatus === 'healthy' ? 'text-green-400' : 'text-red-400'
        }`}>
          {healthStatus === 'healthy' ? 'Healthy' : 'Checking...'}
        </div>
      </motion.div>
    </div>
  );
};

const LoginFormNode = ({ data }: { data: any }) => {
  const {
    formData,
    showPassword,
    loginError,
    isLoginLoading,
    handleSubmit,
    handleInputChange,
    setShowPassword,
    onSwitchToSignup,
    onTryDemo
  } = data;

  return (
    <div className="w-[450px] h-[600px] flex flex-col justify-center items-center p-4">
      <Handle type="target" position={Position.Left} className="w-3 h-3 bg-app-tertiary border-2 border-app-primary" />

      <motion.div
        initial={{ opacity: 0, x: 50 }}
        animate={{ opacity: 1, x: 0 }}
        transition={{ duration: 0.8, delay: 0.3 }}
        className="w-full max-w-md"
      >
        <div className="bg-app-secondary/80 backdrop-blur-xl py-8 px-8 shadow-2xl rounded-3xl border border-app-primary/30 relative overflow-hidden">
          {/* Background Pattern */}
          <div className="absolute inset-0 bg-gradient-to-br from-app-tertiary/5 to-transparent" />
          <div className="absolute top-0 left-0 w-full h-1 bg-gradient-to-r from-transparent via-app-tertiary/50 to-transparent" />

          <div className="relative z-10">
            {/* Form Header */}
            <div className="text-center mb-6">
              <h2 className="text-2xl font-bold text-app-primary mb-2">Welcome Back</h2>
              <p className="text-app-tertiary text-sm">Sign in to your simulation workspace</p>
            </div>

            {/* Login Error */}
            {loginError && (
              <motion.div
                initial={{ opacity: 0, y: -10 }}
                animate={{ opacity: 1, y: 0 }}
                className="mb-4 p-3 bg-red-500/10 border border-red-500/20 rounded-lg flex items-center"
              >
                <AlertCircle className="w-4 h-4 text-red-400 mr-2" />
                <span className="text-red-400 text-sm">{loginError}</span>
              </motion.div>
            )}

            <form className="space-y-6" onSubmit={handleSubmit}>
              {/* Email Field */}
              <div>
                <label htmlFor="email" className="block text-sm font-semibold text-app-secondary mb-2">
                  Email Address
                </label>
                <div className="relative group">
                  <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                    <Mail className="h-5 w-5 text-app-tertiary group-focus-within:text-app-secondary transition-colors duration-200" />
                  </div>
                  <input
                    id="email"
                    name="email"
                    type="email"
                    autoComplete="email"
                    required
                    value={formData.email}
                    onChange={(e) => handleInputChange('email', e.target.value)}
                    className="appearance-none block w-full pl-11 pr-4 py-3 bg-gray-800 border border-gray-600 rounded-lg text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-white/50 focus:border-white/50 text-sm transition-all duration-200 hover:border-gray-500"
                    placeholder="Enter your email address"
                  />
                </div>
              </div>

              {/* Password Field */}
              <div>
                <label htmlFor="password" className="block text-sm font-semibold text-app-secondary mb-2">
                  Password
                </label>
                <div className="relative group">
                  <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                    <Lock className="h-5 w-5 text-app-tertiary group-focus-within:text-app-secondary transition-colors duration-200" />
                  </div>
                  <input
                    id="password"
                    name="password"
                    type={showPassword ? 'text' : 'password'}
                    autoComplete="current-password"
                    required
                    value={formData.password}
                    onChange={(e) => handleInputChange('password', e.target.value)}
                    className="appearance-none block w-full pl-11 pr-11 py-3 bg-gray-800 border border-gray-600 rounded-lg text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-white/50 focus:border-white/50 text-sm transition-all duration-200 hover:border-gray-500"
                    placeholder="Enter your password"
                  />
                  <button
                    type="button"
                    className="absolute inset-y-0 right-0 pr-3 flex items-center group"
                    onClick={() => setShowPassword(!showPassword)}
                  >
                    {showPassword ? (
                      <EyeOff className="h-5 w-5 text-app-tertiary hover:text-app-secondary transition-colors duration-200" />
                    ) : (
                      <Eye className="h-5 w-5 text-app-tertiary hover:text-app-secondary transition-colors duration-200" />
                    )}
                  </button>
                </div>
              </div>

              {/* Remember me & Forgot password */}
              <div className="flex items-center justify-between">
                <div className="flex items-center">
                  <motion.div
                    whileTap={{ scale: 0.95 }}
                    className="relative"
                  >
                    <input
                      id="remember-me"
                      name="remember-me"
                      type="checkbox"
                      checked={formData.remember_me}
                      onChange={(e) => handleInputChange('remember_me', e.target.checked)}
                      className="sr-only"
                    />
                    <motion.div
                      onClick={(e) => {
                        e.preventDefault();
                        e.stopPropagation();
                        console.log('Checkbox clicked, current value:', formData.remember_me, 'new value:', !formData.remember_me);
                        handleInputChange('remember_me', !formData.remember_me);
                      }}
                      className={`w-5 h-5 rounded border-2 cursor-pointer transition-all duration-200 flex items-center justify-center ${
                        formData.remember_me
                          ? 'bg-gray-800 border-gray-400 shadow-md'
                          : 'bg-gray-800 border-gray-400 hover:border-gray-300'
                      }`}
                      whileHover={{ scale: 1.05 }}
                    >
                      {formData.remember_me ? (
                        <motion.div
                          initial={{ scale: 0 }}
                          animate={{ scale: 1 }}
                          transition={{ type: "spring", stiffness: 500, damping: 30 }}
                          className="text-white font-black text-lg leading-none"
                          style={{ fontSize: '16px' }}
                        >
                          ✓
                        </motion.div>
                      ) : null}
                    </motion.div>
                  </motion.div>
                  <label
                    className="ml-3 block text-sm text-app-secondary font-medium cursor-pointer"
                    onClick={(e) => {
                      e.preventDefault();
                      e.stopPropagation();
                      handleInputChange('remember_me', !formData.remember_me);
                    }}
                  >
                    Remember me
                  </label>
                </div>

                <div className="text-sm">
                  <a href="#" className="font-medium text-app-tertiary hover:text-app-secondary transition-colors duration-200">
                    Forgot password?
                  </a>
                </div>
              </div>

              {/* Login Button */}
              <div className="pt-2">
                <motion.button
                  type="submit"
                  disabled={isLoginLoading}
                  whileHover={{ scale: 1.02 }}
                  whileTap={{ scale: 0.98 }}
                  className="group relative w-full flex justify-center items-center py-3 px-6 border border-transparent text-sm font-semibold rounded-xl text-app-primary bg-app-tertiary/30 hover:bg-app-tertiary/40 focus:outline-none focus:ring-2 focus:ring-app-secondary/50 focus:ring-offset-2 focus:ring-offset-app-secondary disabled:opacity-50 disabled:cursor-not-allowed transition-all duration-200 backdrop-blur-sm"
                >
                  {isLoginLoading ? (
                    <div className="w-5 h-5 border-2 border-app-primary border-t-transparent rounded-full animate-spin" />
                  ) : (
                    <>
                      <span>Sign In</span>
                      <ArrowRight className="ml-2 w-4 h-4 group-hover:translate-x-1 transition-transform duration-200" />
                    </>
                  )}
                </motion.button>
              </div>

              {/* Demo Button */}
              <div className="pt-3">
                <motion.button
                  type="button"
                  onClick={onTryDemo}
                  whileHover={{ scale: 1.02 }}
                  whileTap={{ scale: 0.98 }}
                  className="w-full flex justify-center items-center py-3 px-6 border border-app-primary/30 text-sm font-medium rounded-xl text-app-tertiary bg-transparent hover:bg-app-tertiary/10 hover:border-app-secondary/50 hover:text-app-secondary focus:outline-none focus:ring-2 focus:ring-app-secondary/50 focus:ring-offset-2 focus:ring-offset-app-secondary transition-all duration-200"
                >
                  <Layers className="mr-2 w-4 h-4" />
                  Try Demo (No Login Required)
                </motion.button>
              </div>
            </form>

            {/* Sign up link */}
            <div className="mt-6 pt-4">
              <div className="relative flex items-center">
                <div className="flex-grow border-t border-app-primary/30"></div>
                <span className="px-4 text-sm text-app-tertiary font-medium bg-app-secondary/80">New to SystemSim?</span>
                <div className="flex-grow border-t border-app-primary/30"></div>
              </div>

              <div className="mt-4 text-center">
                <motion.button
                  type="button"
                  onClick={onSwitchToSignup}
                  whileHover={{ scale: 1.05 }}
                  whileTap={{ scale: 0.95 }}
                  className="inline-flex items-center font-medium text-app-tertiary hover:text-app-secondary transition-colors duration-200 text-sm"
                >
                  <span>Create your account</span>
                  <ArrowRight className="ml-1 w-3 h-3" />
                </motion.button>
              </div>
            </div>
          </div>
        </div>
      </motion.div>
    </div>
  );
};

export const LoginPage: React.FC<LoginPageProps> = ({ onSuccess, onSwitchToSignup, onTryDemo }) => {
  const { login, isLoginLoading, loginError, clearLoginError } = useAuthStore();
  const { healthStatus, error, cleanup } = useAuthHealthCheck(); // WebSocket only
  const [showCertHelper, setShowCertHelper] = useState(false);

  const [formData, setFormData] = useState<LoginRequest>({
    email: '',
    password: '',
    remember_me: false,
  });

  const [showPassword, setShowPassword] = useState(false);
  const [windowSize, setWindowSize] = useState({ width: window.innerWidth, height: window.innerHeight });

  // Update window size on resize
  useEffect(() => {
    const handleResize = () => {
      setWindowSize({ width: window.innerWidth, height: window.innerHeight });
    };

    window.addEventListener('resize', handleResize);
    return () => window.removeEventListener('resize', handleResize);
  }, []);

  // Show certificate helper for connection issues
  useEffect(() => {
    if (healthStatus === 'disconnected' && error && (
      error.includes('Connection failed') ||
      error.includes('Certificate issue') ||
      error.includes('SSL/TLS error') ||
      error.includes('ERR_CERT_AUTHORITY_INVALID')
    )) {
      setShowCertHelper(true);
    }
  }, [healthStatus, error]);

  // Handle form submission
  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    clearLoginError();

    // Submit login
    const success = await login(formData);
    if (success) {
      // Cleanup auth health WebSocket before navigation
      cleanup();
      onSuccess();
    }
  };

  const handleInputChange = (field: keyof LoginRequest, value: string | boolean) => {
    setFormData(prev => ({ ...prev, [field]: value }));
  };

  // React Flow setup
  const nodeTypes = {
    systemSimLogo: SystemSimLogoNode,
    authSystem: AuthSystemNode,
    loginForm: LoginFormNode,
  };

  // Calculate positions based on window size
  const calculatePositions = () => {
    const { width, height } = windowSize;

    // Node dimensions (from our fixed sizes)
    const logoWidth = 400, logoHeight = 500;
    const authWidth = 100, authHeight = 80;
    const formWidth = 450, formHeight = 600;

    // Calculate spacing between columns
    const leftSpacing = 60; // Spacing between logo and auth icon
    const rightSpacing = 80; // Slightly more spacing between auth icon and form

    // Vertical centering
    const logoY = (height - logoHeight) / 2;
    const authY = (height - authHeight) / 2;
    const formY = (height - formHeight) / 2;

    // Horizontal positioning - compact layout
    const logoX = (width - (logoWidth + leftSpacing + authWidth + rightSpacing + formWidth)) / 2;
    const authX = logoX + logoWidth + leftSpacing;
    const formX = authX + authWidth + rightSpacing;

    return {
      logo: { x: logoX, y: logoY },
      auth: { x: authX, y: authY },
      form: { x: formX, y: formY }
    };
  };

  const positions = calculatePositions();

  const initialNodes: Node[] = [
    {
      id: 'systemsim-logo',
      type: 'systemSimLogo',
      position: positions.logo,
      data: {},
      draggable: true,
      selectable: true,
    },
    {
      id: 'auth-system',
      type: 'authSystem',
      position: positions.auth,
      data: { isLoginLoading, loginError, healthStatus },
      draggable: true,
      selectable: true,
    },
    {
      id: 'login-form',
      type: 'loginForm',
      position: positions.form,
      data: {
        formData,
        showPassword,
        loginError,
        isLoginLoading,
        handleSubmit,
        handleInputChange,
        setShowPassword,
        onSwitchToSignup,
        onTryDemo,
      },
      draggable: true,
      selectable: true,
    },
  ];

  const initialEdges: Edge[] = [
    {
      id: 'logo-to-auth',
      source: 'systemsim-logo',
      target: 'auth-system',
      type: 'straight',
      style: { stroke: '#ffffff', strokeWidth: 2, opacity: 0.8 },
      animated: isLoginLoading,
    },
    {
      id: 'auth-to-form',
      source: 'auth-system',
      target: 'login-form',
      type: 'straight',
      style: { stroke: '#ffffff', strokeWidth: 2, opacity: 0.8 },
      animated: isLoginLoading,
    },
  ];

  const [nodes, setNodes, onNodesChange] = useNodesState(initialNodes);
  const [edges, setEdges, onEdgesChange] = useEdgesState(initialEdges);

  // Update nodes when state changes
  useEffect(() => {
    setNodes((nds) =>
      nds.map((node) => {
        if (node.id === 'auth-system') {
          return {
            ...node,
            data: { isLoginLoading, loginError, healthStatus },
          };
        }
        if (node.id === 'login-form') {
          return {
            ...node,
            data: {
              formData,
              showPassword,
              loginError,
              isLoginLoading,
              handleSubmit,
              handleInputChange,
              setShowPassword,
              onSwitchToSignup,
              onTryDemo,
            },
          };
        }
        return node;
      })
    );
  }, [isLoginLoading, loginError, healthStatus, formData, showPassword]);

  // Update node positions when window size changes
  useEffect(() => {
    const newPositions = calculatePositions();
    setNodes((nds) =>
      nds.map((node) => {
        if (node.id === 'systemsim-logo') {
          return { ...node, position: newPositions.logo };
        }
        if (node.id === 'auth-system') {
          return { ...node, position: newPositions.auth };
        }
        if (node.id === 'login-form') {
          return { ...node, position: newPositions.form };
        }
        return node;
      })
    );
  }, [windowSize]);

  // Update edges animation
  useEffect(() => {
    setEdges((eds) =>
      eds.map((edge) => ({
        ...edge,
        animated: isLoginLoading,
      }))
    );
  }, [isLoginLoading]);

  return (
    <div className="h-screen bg-app-primary flex overflow-hidden relative">
      {/* Background Pattern - Eraser.io Style */}
      <div className="absolute inset-0 bg-grid-pattern opacity-20" />

      {/* Enhanced Animated Background Elements */}
      <div className="absolute inset-0 overflow-hidden">
        {/* Floating geometric shapes */}
        <motion.div
          animate={{
            x: [0, 100, 0],
            y: [0, -50, 0],
            rotate: [0, 180, 360]
          }}
          transition={{ duration: 20, repeat: Infinity, ease: "linear" }}
          className="absolute top-1/4 left-1/4 w-2 h-2 bg-app-tertiary rounded-full opacity-20"
        />
        <motion.div
          animate={{
            x: [0, -80, 0],
            y: [0, 60, 0],
            rotate: [0, -180, -360]
          }}
          transition={{ duration: 25, repeat: Infinity, ease: "linear" }}
          className="absolute top-3/4 right-1/3 w-1 h-1 bg-app-tertiary rounded-full opacity-30"
        />
        <motion.div
          animate={{
            x: [0, 60, 0],
            y: [0, -80, 0],
            scale: [1, 1.2, 1]
          }}
          transition={{ duration: 15, repeat: Infinity, ease: "linear" }}
          className="absolute bottom-1/4 left-1/2 w-1.5 h-1.5 bg-app-tertiary rounded-full opacity-25"
        />

        {/* Additional floating elements */}
        <motion.div
          animate={{
            x: [0, -120, 0],
            y: [0, 80, 0],
            rotate: [0, 90, 180, 270, 360]
          }}
          transition={{ duration: 30, repeat: Infinity, ease: "linear" }}
          className="absolute top-1/3 right-1/4 w-4 h-4 border border-app-tertiary/20 rounded-lg opacity-15"
        />
        <motion.div
          animate={{
            x: [0, 90, 0],
            y: [0, -60, 0],
            scale: [1, 1.5, 1]
          }}
          transition={{ duration: 18, repeat: Infinity, ease: "linear" }}
          className="absolute bottom-1/3 left-1/3 w-3 h-3 bg-app-secondary/20 rounded-full opacity-25"
        />
        <motion.div
          animate={{
            x: [0, -70, 0],
            y: [0, 40, 0],
            rotate: [0, -90, -180, -270, -360]
          }}
          transition={{ duration: 22, repeat: Infinity, ease: "linear" }}
          className="absolute top-2/3 left-1/5 w-2 h-6 bg-app-tertiary/15 rounded-full opacity-20"
        />
        <motion.div
          animate={{
            x: [0, 110, 0],
            y: [0, -90, 0],
            scale: [1, 0.8, 1.3, 1]
          }}
          transition={{ duration: 28, repeat: Infinity, ease: "linear" }}
          className="absolute bottom-1/5 right-1/5 w-5 h-1 bg-app-secondary/15 rounded-full opacity-20"
        />

        {/* Floating code-like elements */}
        <motion.div
          animate={{
            x: [0, -50, 0],
            y: [0, 30, 0],
            opacity: [0.1, 0.3, 0.1]
          }}
          transition={{ duration: 12, repeat: Infinity, ease: "linear" }}
          className="absolute top-1/5 left-2/3 text-app-tertiary/20 text-xs font-mono"
        >
          {'{ api: "ready" }'}
        </motion.div>
        <motion.div
          animate={{
            x: [0, 40, 0],
            y: [0, -20, 0],
            opacity: [0.1, 0.25, 0.1]
          }}
          transition={{ duration: 16, repeat: Infinity, ease: "linear", delay: 2 }}
          className="absolute bottom-1/3 left-1/6 text-app-tertiary/15 text-xs font-mono"
        >
          {'<system/>'}
        </motion.div>
        <motion.div
          animate={{
            x: [0, -30, 0],
            y: [0, 50, 0],
            opacity: [0.1, 0.2, 0.1]
          }}
          transition={{ duration: 14, repeat: Infinity, ease: "linear", delay: 4 }}
          className="absolute top-3/5 right-1/3 text-app-tertiary/20 text-xs font-mono"
        >
          {'[auth]'}
        </motion.div>

        {/* Subtle gradient orbs */}
        <motion.div
          animate={{
            scale: [1, 1.2, 1],
            opacity: [0.05, 0.15, 0.05]
          }}
          transition={{ duration: 8, repeat: Infinity, ease: "easeInOut" }}
          className="absolute top-1/6 right-1/6 w-32 h-32 bg-gradient-to-br from-app-tertiary/10 to-transparent rounded-full blur-xl"
        />
        <motion.div
          animate={{
            scale: [1, 1.3, 1],
            opacity: [0.03, 0.12, 0.03]
          }}
          transition={{ duration: 10, repeat: Infinity, ease: "easeInOut", delay: 2 }}
          className="absolute bottom-1/4 left-1/8 w-40 h-40 bg-gradient-to-tr from-app-secondary/8 to-transparent rounded-full blur-2xl"
        />
      </div>

      {/* React Flow Container - Fixed Canvas */}
      <ReactFlowProvider>
        <div className="absolute inset-0 w-full h-full">
          <ReactFlow
            nodes={nodes}
            edges={edges}
            onNodesChange={onNodesChange}
            onEdgesChange={onEdgesChange}
            nodeTypes={nodeTypes}
            nodesDraggable={true}
            nodesConnectable={false}
            elementsSelectable={true}
            panOnDrag={false}
            zoomOnScroll={false}
            zoomOnPinch={false}
            zoomOnDoubleClick={false}
            preventScrolling={false}
            className="bg-transparent"
            proOptions={{ hideAttribution: true }}
            defaultViewport={{ x: 0, y: 0, zoom: 1 }}
            minZoom={1}
            maxZoom={1}
            fitView={false}
          />
        </div>
      </ReactFlowProvider>

      {/* Certificate Helper Modal */}
      <CertificateHelper
        isVisible={showCertHelper}
        onClose={() => setShowCertHelper(false)}
        onRetry={() => window.location.reload()}
      />
    </div>
  );
};
