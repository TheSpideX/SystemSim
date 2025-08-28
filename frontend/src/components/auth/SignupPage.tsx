import React, { useState, useEffect } from 'react';
import { motion } from 'framer-motion';
import {
  Zap, Mail, Lock, ArrowRight, Eye, EyeOff, User, Building,
  Activity, Layers, Network, CheckCircle, AlertCircle, Shield
} from 'lucide-react';
import { useAuthStore } from '../../store/authStore';
import type { RegisterRequest } from '../../store/authStore';
import { useAuthHealthCheck } from '../../hooks/useAuthHealthCheck';
import { CertificateHelper } from '../ui/CertificateHelper';

interface SignupPageProps {
  onSuccess: () => void;
  onSwitchToLogin: () => void;
}

export const SignupPage: React.FC<SignupPageProps> = ({ onSuccess, onSwitchToLogin }) => {
  const { register, isRegisterLoading, registerError, clearRegisterError } = useAuthStore();
  const { healthStatus, lastChecked, error, isConnected, cleanup } = useAuthHealthCheck(); // WebSocket only
  const [showCertHelper, setShowCertHelper] = useState(false);

  const [formData, setFormData] = useState<RegisterRequest>({
    email: '',
    password: '',
    first_name: '',
    last_name: '',
    company: '',
  });

  const [confirmPassword, setConfirmPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [showConfirmPassword, setShowConfirmPassword] = useState(false);

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

  // Password strength validation
  const validatePassword = (password: string) => {
    const errors: string[] = [];
    if (password.length < 8) errors.push('At least 8 characters');
    if (!/[A-Z]/.test(password)) errors.push('One uppercase letter');
    if (!/[a-z]/.test(password)) errors.push('One lowercase letter');
    if (!/\d/.test(password)) errors.push('One number');
    if (!/[!@#$%^&*(),.?":{}|<>]/.test(password)) errors.push('One special character');
    return errors;
  };

  const passwordErrors = validatePassword(formData.password);
  const isPasswordValid = passwordErrors.length === 0;
  const doPasswordsMatch = formData.password === confirmPassword;

  // Handle form submission
  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    clearRegisterError();

    // Basic validation
    if (!isPasswordValid || !doPasswordsMatch) {
      return;
    }

    // Submit registration
    const success = await register(formData);
    if (success) {
      // Cleanup auth health WebSocket before navigation
      cleanup();
      onSuccess();
    }
  };

  const handleInputChange = (field: keyof RegisterRequest, value: string) => {
    setFormData(prev => ({ ...prev, [field]: value }));
  };

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
          {'{ user: "new" }'}
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
          {'<signup/>'}
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
          {'[register]'}
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

      {/* Main Content Container */}
      <div className="flex max-w-6xl w-full mx-auto relative z-10 items-center">
        {/* Left Column - Logo and Tagline */}
        <div className="flex-1 flex flex-col justify-center items-center pr-12">
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
            className="text-center max-w-md"
          >
            <h1 className="text-5xl font-bold text-app-primary mb-4">SystemSim</h1>
            <div className="flex items-center justify-center gap-3 mb-6">
              <div className="w-12 h-0.5 bg-app-tertiary rounded-full" />
              <Layers className="w-5 h-5 text-app-tertiary" />
              <div className="w-12 h-0.5 bg-app-tertiary rounded-full" />
            </div>
            <p className="text-xl text-app-secondary font-medium mb-3">
              Join the System Design Revolution
            </p>
            <p className="text-app-tertiary text-base leading-relaxed">
              Create your account to start building, testing, and optimizing distributed systems
              with our advanced simulation platform.
            </p>

            {/* Feature highlights */}
            <div className="mt-8 space-y-3">
              <div className="flex items-center justify-center gap-3 text-app-tertiary">
                <Activity className="w-4 h-4" />
                <span className="text-sm">Real-time performance monitoring</span>
              </div>
              <div className="flex items-center justify-center gap-3 text-app-tertiary">
                <Network className="w-4 h-4" />
                <span className="text-sm">Collaborative design workspace</span>
              </div>
              <div className="flex items-center justify-center gap-3 text-app-tertiary">
                <Layers className="w-4 h-4" />
                <span className="text-sm">Advanced simulation engine</span>
              </div>
            </div>
          </motion.div>
        </div>

        {/* Center Auth System Icon with Connection Lines */}
        <div className="flex flex-col items-center justify-center px-8 relative">
          {/* Simple visible white lines */}
          <div className="absolute inset-0 flex items-center z-0">
            {/* Left line */}
            <div className="h-px bg-white flex-1 mx-8" style={{ opacity: 0.6 }} />

            {/* Center space for icon */}
            <div className="w-16" />

            {/* Right line */}
            <div className="h-px bg-white flex-1 mx-8" style={{ opacity: 0.6 }} />
          </div>

          {/* Animated particles during signup */}
          {isRegisterLoading && (
            <>
              {/* Left to center particle */}
              <motion.div
                className="absolute w-2 h-2 bg-white rounded-full z-20"
                style={{ left: '10%', top: '50%', transform: 'translateY(-50%)' }}
                animate={{ left: ['10%', '50%'] }}
                transition={{
                  duration: 1.5,
                  repeat: Infinity,
                  ease: 'linear',
                }}
              />
              {/* Center to right particle */}
              <motion.div
                className="absolute w-2 h-2 bg-white rounded-full z-20"
                style={{ left: '50%', top: '50%', transform: 'translateY(-50%)' }}
                animate={{ left: ['50%', '90%'] }}
                transition={{
                  duration: 1.5,
                  repeat: Infinity,
                  ease: 'linear',
                  delay: 0.75,
                }}
              />
            </>
          )}



          {/* Simple Auth System Icon */}
          <motion.div
            animate={isRegisterLoading || healthStatus === 'checking' ? {
              scale: [1, 1.1, 1]
            } : {}}
            transition={{ duration: 1.5, repeat: (isRegisterLoading || healthStatus === 'checking') ? Infinity : 0 }}
            className="w-16 h-16 bg-app-secondary/80 backdrop-blur-xl rounded-2xl border border-app-primary/30 flex items-center justify-center shadow-xl relative z-10"
          >
            <Shield className="w-8 h-8 text-app-tertiary" />

            {/* Health-aware status indicator */}
            <motion.div
              className={`absolute -top-1 -right-1 w-4 h-4 rounded-full border-2 border-app-secondary ${
                registerError ? 'bg-red-400' :
                isRegisterLoading ? 'bg-yellow-400' :
                healthStatus === 'checking' ? 'bg-blue-400' :
                healthStatus === 'healthy' ? 'bg-green-400' : 'bg-red-400'
              }`}
              animate={healthStatus === 'checking' ? { scale: [1, 1.2, 1] } : {}}
              transition={{ duration: 1, repeat: healthStatus === 'checking' ? Infinity : 0 }}
            />
          </motion.div>

          {/* Auth System Label with Health Status */}
          <motion.div
            className="mt-4 text-center relative z-10"
            animate={isRegisterLoading ? { opacity: [1, 0.7, 1] } : {}}
            transition={{ duration: 2, repeat: isRegisterLoading ? Infinity : 0 }}
          >
            <div className="text-sm font-bold text-app-secondary">Auth System</div>
            <div className={`text-xs font-medium ${
              registerError ? 'text-red-400' :
              isRegisterLoading ? 'text-yellow-400' :
              healthStatus === 'checking' ? 'text-blue-400' :
              healthStatus === 'healthy' ? 'text-green-400' : 'text-red-400'
            }`}>
              {isRegisterLoading ? 'Creating Account...' :
               registerError ? 'Registration Failed' :
               healthStatus === 'checking' ? 'Health Check...' :
               healthStatus === 'healthy' ? 'Healthy' :
               healthStatus === 'disconnected' ? 'Disconnected' : 'Unhealthy'}
            </div>
          </motion.div>
        </div>

        {/* Right Column - Signup Form */}
        <div className="flex-1 flex flex-col justify-center items-center pl-12">
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
                  <h2 className="text-2xl font-bold text-app-primary mb-2">Create Account</h2>
                  <p className="text-app-tertiary text-sm">Start your system design journey</p>
                </div>

                {/* Registration Error */}
                {registerError && (
                  <motion.div
                    initial={{ opacity: 0, y: -10 }}
                    animate={{ opacity: 1, y: 0 }}
                    className="mb-4 p-3 bg-red-500/10 border border-red-500/20 rounded-lg flex items-center"
                  >
                    <AlertCircle className="w-4 h-4 text-red-400 mr-2" />
                    <span className="text-red-400 text-sm">{registerError}</span>
                  </motion.div>
                )}

                <form className="space-y-3" onSubmit={handleSubmit}>
                  {/* Name Fields */}
                  <div className="grid grid-cols-2 gap-3">
                    <div>
                      <label htmlFor="first_name" className="block text-sm font-semibold text-app-secondary mb-1">
                        First Name
                      </label>
                      <div className="relative group">
                        <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                          <User className="h-5 w-5 text-app-tertiary group-focus-within:text-app-secondary transition-colors duration-200" />
                        </div>
                        <input
                          id="first_name"
                          name="first_name"
                          type="text"
                          required
                          value={formData.first_name}
                          onChange={(e) => handleInputChange('first_name', e.target.value)}
                          className="appearance-none block w-full pl-11 pr-4 py-2.5 bg-app-tertiary/20 border border-app-primary/30 rounded-xl text-app-primary placeholder-app-tertiary focus:outline-none focus:ring-2 focus:ring-app-secondary/50 focus:border-app-secondary/50 text-sm transition-all duration-200 hover:border-app-secondary/30"
                          placeholder="John"
                        />
                      </div>
                    </div>

                    <div>
                      <label htmlFor="last_name" className="block text-sm font-semibold text-app-secondary mb-1">
                        Last Name
                      </label>
                      <div className="relative group">
                        <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                          <User className="h-5 w-5 text-app-tertiary group-focus-within:text-app-secondary transition-colors duration-200" />
                        </div>
                        <input
                          id="last_name"
                          name="last_name"
                          type="text"
                          required
                          value={formData.last_name}
                          onChange={(e) => handleInputChange('last_name', e.target.value)}
                          className="appearance-none block w-full pl-11 pr-4 py-2.5 bg-app-tertiary/20 border border-app-primary/30 rounded-xl text-app-primary placeholder-app-tertiary focus:outline-none focus:ring-2 focus:ring-app-secondary/50 focus:border-app-secondary/50 text-sm transition-all duration-200 hover:border-app-secondary/30"
                          placeholder="Doe"
                        />
                      </div>
                    </div>
                  </div>

                  {/* Email Field */}
                  <div>
                    <label htmlFor="email" className="block text-sm font-semibold text-app-secondary mb-1">
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
                        className="appearance-none block w-full pl-11 pr-4 py-2.5 bg-app-tertiary/20 border border-app-primary/30 rounded-xl text-app-primary placeholder-app-tertiary focus:outline-none focus:ring-2 focus:ring-app-secondary/50 focus:border-app-secondary/50 text-sm transition-all duration-200 hover:border-app-secondary/30"
                        placeholder="Enter your email address"
                      />
                    </div>
                  </div>

                  {/* Company Field (Optional) */}
                  <div>
                    <label htmlFor="company" className="block text-sm font-semibold text-app-secondary mb-1">
                      Company <span className="text-app-tertiary text-xs">(Optional)</span>
                    </label>
                    <div className="relative group">
                      <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                        <Building className="h-5 w-5 text-app-tertiary group-focus-within:text-app-secondary transition-colors duration-200" />
                      </div>
                      <input
                        id="company"
                        name="company"
                        type="text"
                        value={formData.company}
                        onChange={(e) => handleInputChange('company', e.target.value)}
                        className="appearance-none block w-full pl-11 pr-4 py-2.5 bg-app-tertiary/20 border border-app-primary/30 rounded-xl text-app-primary placeholder-app-tertiary focus:outline-none focus:ring-2 focus:ring-app-secondary/50 focus:border-app-secondary/50 text-sm transition-all duration-200 hover:border-app-secondary/30"
                        placeholder="Your Company"
                      />
                    </div>
                  </div>

                  {/* Password Field */}
                  <div>
                    <label htmlFor="password" className="block text-sm font-semibold text-app-secondary mb-1">
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
                        autoComplete="new-password"
                        required
                        value={formData.password}
                        onChange={(e) => handleInputChange('password', e.target.value)}
                        className="appearance-none block w-full pl-11 pr-11 py-2.5 bg-app-tertiary/20 border border-app-primary/30 rounded-xl text-app-primary placeholder-app-tertiary focus:outline-none focus:ring-2 focus:ring-app-secondary/50 focus:border-app-secondary/50 text-sm transition-all duration-200 hover:border-app-secondary/30"
                        placeholder="Create a strong password"
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

                    {/* Password Requirements */}
                    {formData.password && (
                      <div className="mt-2 space-y-1">
                        {passwordErrors.map((error, index) => (
                          <div key={index} className="flex items-center text-xs">
                            <div className="w-1 h-1 bg-red-400 rounded-full mr-2" />
                            <span className="text-red-400">{error}</span>
                          </div>
                        ))}
                        {isPasswordValid && (
                          <div className="flex items-center text-xs">
                            <CheckCircle className="w-3 h-3 text-green-400 mr-2" />
                            <span className="text-green-400">Password meets all requirements</span>
                          </div>
                        )}
                      </div>
                    )}
                  </div>

                  {/* Confirm Password Field */}
                  <div>
                    <label htmlFor="confirmPassword" className="block text-sm font-semibold text-app-secondary mb-1">
                      Confirm Password
                    </label>
                    <div className="relative group">
                      <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                        <Lock className="h-5 w-5 text-app-tertiary group-focus-within:text-app-secondary transition-colors duration-200" />
                      </div>
                      <input
                        id="confirmPassword"
                        name="confirmPassword"
                        type={showConfirmPassword ? 'text' : 'password'}
                        autoComplete="new-password"
                        required
                        value={confirmPassword}
                        onChange={(e) => setConfirmPassword(e.target.value)}
                        className="appearance-none block w-full pl-11 pr-11 py-2.5 bg-app-tertiary/20 border border-app-primary/30 rounded-xl text-app-primary placeholder-app-tertiary focus:outline-none focus:ring-2 focus:ring-app-secondary/50 focus:border-app-secondary/50 text-sm transition-all duration-200 hover:border-app-secondary/30"
                        placeholder="Confirm your password"
                      />
                      <button
                        type="button"
                        className="absolute inset-y-0 right-0 pr-3 flex items-center group"
                        onClick={() => setShowConfirmPassword(!showConfirmPassword)}
                      >
                        {showConfirmPassword ? (
                          <EyeOff className="h-5 w-5 text-app-tertiary hover:text-app-secondary transition-colors duration-200" />
                        ) : (
                          <Eye className="h-5 w-5 text-app-tertiary hover:text-app-secondary transition-colors duration-200" />
                        )}
                      </button>
                    </div>

                    {/* Password Match Indicator */}
                    {confirmPassword && (
                      <div className="mt-1">
                        {doPasswordsMatch ? (
                          <div className="flex items-center text-xs">
                            <CheckCircle className="w-3 h-3 text-green-400 mr-2" />
                            <span className="text-green-400">Passwords match</span>
                          </div>
                        ) : (
                          <div className="flex items-center text-xs">
                            <div className="w-1 h-1 bg-red-400 rounded-full mr-2" />
                            <span className="text-red-400">Passwords do not match</span>
                          </div>
                        )}
                      </div>
                    )}
                  </div>

                  {/* Create Account Button */}
                  <div className="pt-2">
                    <motion.button
                      type="submit"
                      disabled={isRegisterLoading || !isPasswordValid || !doPasswordsMatch}
                      whileHover={{ scale: 1.02 }}
                      whileTap={{ scale: 0.98 }}
                      className="group relative w-full flex justify-center items-center py-3 px-6 border border-transparent text-sm font-semibold rounded-xl text-app-primary bg-app-tertiary/30 hover:bg-app-tertiary/40 focus:outline-none focus:ring-2 focus:ring-app-secondary/50 focus:ring-offset-2 focus:ring-offset-app-secondary disabled:opacity-50 disabled:cursor-not-allowed transition-all duration-200 backdrop-blur-sm"
                    >
                      {isRegisterLoading ? (
                        <div className="w-5 h-5 border-2 border-app-primary border-t-transparent rounded-full animate-spin" />
                      ) : (
                        <>
                          <span>Create Account</span>
                          <ArrowRight className="ml-2 w-4 h-4 group-hover:translate-x-1 transition-transform duration-200" />
                        </>
                      )}
                    </motion.button>
                  </div>
                </form>

                {/* Sign in link */}
                <div className="mt-6 pt-4">
                  <div className="relative">
                    <div className="absolute inset-0 flex items-center">
                      <div className="w-full border-t border-app-primary/20" />
                    </div>
                    <div className="relative flex justify-center text-sm">
                      <span className="px-3 bg-app-secondary/80 text-app-tertiary font-medium">Already have an account?</span>
                    </div>
                  </div>

                  <div className="mt-4 text-center">
                    <motion.button
                      type="button"
                      onClick={onSwitchToLogin}
                      whileHover={{ scale: 1.05 }}
                      whileTap={{ scale: 0.95 }}
                      className="inline-flex items-center font-medium text-app-tertiary hover:text-app-secondary transition-colors duration-200 text-sm"
                    >
                      <span>Sign in to your account</span>
                      <ArrowRight className="ml-1 w-3 h-3" />
                    </motion.button>
                  </div>
                </div>
              </div>
            </div>
          </motion.div>
        </div>
      </div>

      {/* Certificate Helper Modal */}
      <CertificateHelper
        isVisible={showCertHelper}
        onClose={() => setShowCertHelper(false)}
        onRetry={() => window.location.reload()}
      />
    </div>
  );
};
