import React, { useState } from 'react';
import { motion } from 'framer-motion';
import {
  ArrowLeft, Zap, Settings, Share2, Download, Maximize2, Minimize2,
  Sun, Moon, User, LogOut, Save, Check, Loader2, Edit3
} from 'lucide-react';
import { useThemeStore } from '../../store/themeStore';
import { useUIStore } from '../../store/uiStore';
import { ModeSwitcher } from './ModeSwitcher';
import toast from 'react-hot-toast';

interface HeaderProps {
  onBack: () => void;
}

export const Header: React.FC<HeaderProps> = ({ onBack }) => {
  const { theme, toggleTheme } = useThemeStore();
  const { 
    projectTitle, 
    saveStatus, 
    setSaveStatus, 
    setProjectTitle,
    isFocusMode,
    toggleFocusMode 
  } = useUIStore();
  
  const [isEditingTitle, setIsEditingTitle] = useState(false);
  const [tempTitle, setTempTitle] = useState(projectTitle);
  const [showUserMenu, setShowUserMenu] = useState(false);

  const handleSave = async () => {
    setSaveStatus('saving');
    // Simulate save operation
    await new Promise(resolve => setTimeout(resolve, 1000));
    setSaveStatus('saved');
    toast.success('Project saved successfully');
  };

  const handleTitleEdit = () => {
    setIsEditingTitle(true);
    setTempTitle(projectTitle);
  };

  const handleTitleSave = () => {
    setProjectTitle(tempTitle);
    setIsEditingTitle(false);
    toast.success('Project renamed');
  };

  const handleTitleCancel = () => {
    setTempTitle(projectTitle);
    setIsEditingTitle(false);
  };

  const handleShare = () => {
    navigator.clipboard.writeText(window.location.href);
    toast.success('Project link copied to clipboard');
  };

  const handleExport = () => {
    toast.success('Export started - check downloads');
  };

  const getSaveIcon = () => {
    switch (saveStatus) {
      case 'saving':
        return <Loader2 className="w-4 h-4 animate-spin" />;
      case 'saved':
        return <Check className="w-4 h-4" />;
      default:
        return <Save className="w-4 h-4" />;
    }
  };

  const getSaveText = () => {
    switch (saveStatus) {
      case 'saving':
        return 'Saving...';
      case 'saved':
        return 'Saved';
      default:
        return 'Save';
    }
  };

  return (
    <motion.header
      initial={{ y: -20, opacity: 0 }}
      animate={{ y: 0, opacity: 1 }}
      className="bg-white dark:bg-black border-b border-gray-200 dark:border-gray-800 px-4 py-3 flex items-center z-50 shadow-sm"
    >
      {/* Left Section */}
      <div className="flex items-center space-x-4 flex-1">
        <motion.button
          whileHover={{ scale: 1.05 }}
          whileTap={{ scale: 0.95 }}
          onClick={onBack}
          className="p-2 text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white hover:bg-gray-100 dark:hover:bg-gray-800 rounded-lg transition-all duration-200"
        >
          <ArrowLeft className="w-5 h-5" />
        </motion.button>
        
        <div className="flex items-center space-x-3">
          <div className="w-8 h-8 bg-gray-100 dark:bg-gray-900 rounded-lg flex items-center justify-center border border-gray-200 dark:border-gray-700">
            <Zap className="w-5 h-5 text-gray-900 dark:text-white" />
          </div>
          <div>
            <h1 className="text-gray-900 dark:text-white font-semibold">System Design Simulator</h1>
            <div className="flex items-center space-x-2">
              {isEditingTitle ? (
                <div className="flex items-center space-x-2">
                  <input
                    type="text"
                    value={tempTitle}
                    onChange={(e) => setTempTitle(e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === 'Enter') handleTitleSave();
                      if (e.key === 'Escape') handleTitleCancel();
                    }}
                    className="text-sm bg-gray-100 dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded px-2 py-1 text-gray-700 dark:text-gray-300 focus:outline-none focus:ring-2 focus:ring-blue-500"
                    autoFocus
                  />
                  <button
                    onClick={handleTitleSave}
                    className="text-green-600 hover:text-green-700"
                  >
                    <Check className="w-4 h-4" />
                  </button>
                </div>
              ) : (
                <button
                  onClick={handleTitleEdit}
                  className="flex items-center space-x-1 text-gray-500 dark:text-gray-500 text-sm hover:text-gray-700 dark:hover:text-gray-300 group"
                >
                  <span>{projectTitle}</span>
                  <Edit3 className="w-3 h-3 opacity-0 group-hover:opacity-100 transition-opacity" />
                </button>
              )}
            </div>
          </div>
        </div>
      </div>

      {/* Center Section - Mode Switcher */}
      <div className="flex-shrink-0">
        <ModeSwitcher />
      </div>

      {/* Right Section */}
      <div className="flex items-center space-x-2 flex-1 justify-end">
        {/* Theme is locked to dark mode for consistent design */}

        {/* Focus Mode Toggle */}
        <motion.button
          whileHover={{ scale: 1.05 }}
          whileTap={{ scale: 0.95 }}
          onClick={toggleFocusMode}
          className={`p-2 rounded-lg transition-all duration-200 ${
            isFocusMode
              ? 'bg-blue-100 dark:bg-blue-900 text-blue-600 dark:text-blue-400'
              : 'text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white hover:bg-gray-100 dark:hover:bg-gray-800'
          }`}
          title="Toggle focus mode"
        >
          {isFocusMode ? <Minimize2 className="w-5 h-5" /> : <Maximize2 className="w-5 h-5" />}
        </motion.button>

        {/* Save Button */}
        <motion.button
          whileHover={{ scale: 1.05 }}
          whileTap={{ scale: 0.95 }}
          onClick={handleSave}
          disabled={saveStatus === 'saving'}
          className={`flex items-center space-x-2 px-3 py-2 rounded-lg text-sm font-medium transition-all duration-200 ${
            saveStatus === 'saved'
              ? 'bg-green-100 dark:bg-green-900 text-green-700 dark:text-green-300'
              : saveStatus === 'saving'
              ? 'bg-gray-100 dark:bg-gray-800 text-gray-500 dark:text-gray-400 cursor-not-allowed'
              : 'bg-blue-100 dark:bg-blue-900 text-blue-700 dark:text-blue-300 hover:bg-blue-200 dark:hover:bg-blue-800'
          }`}
        >
          {getSaveIcon()}
          <span>{getSaveText()}</span>
        </motion.button>

        {/* Share Button */}
        <motion.button
          whileHover={{ scale: 1.05 }}
          whileTap={{ scale: 0.95 }}
          onClick={handleShare}
          className="p-2 text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white hover:bg-gray-100 dark:hover:bg-gray-800 rounded-lg transition-all duration-200"
          title="Share project"
        >
          <Share2 className="w-5 h-5" />
        </motion.button>

        {/* Export Button */}
        <motion.button
          whileHover={{ scale: 1.05 }}
          whileTap={{ scale: 0.95 }}
          onClick={handleExport}
          className="p-2 text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white hover:bg-gray-100 dark:hover:bg-gray-800 rounded-lg transition-all duration-200"
          title="Export project"
        >
          <Download className="w-5 h-5" />
        </motion.button>

        {/* Settings Button */}
        <motion.button
          whileHover={{ scale: 1.05 }}
          whileTap={{ scale: 0.95 }}
          className="p-2 text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white hover:bg-gray-100 dark:hover:bg-gray-800 rounded-lg transition-all duration-200"
          title="Settings"
        >
          <Settings className="w-5 h-5" />
        </motion.button>

        {/* User Menu */}
        <div className="relative">
          <motion.button
            whileHover={{ scale: 1.05 }}
            whileTap={{ scale: 0.95 }}
            onClick={() => setShowUserMenu(!showUserMenu)}
            className="w-8 h-8 bg-gray-200 dark:bg-gray-700 rounded-full flex items-center justify-center text-gray-600 dark:text-gray-400 hover:bg-gray-300 dark:hover:bg-gray-600 transition-all duration-200"
          >
            <User className="w-4 h-4" />
          </motion.button>

          {showUserMenu && (
            <motion.div
              initial={{ opacity: 0, scale: 0.95, y: -10 }}
              animate={{ opacity: 1, scale: 1, y: 0 }}
              exit={{ opacity: 0, scale: 0.95, y: -10 }}
              className="absolute right-0 top-full mt-2 w-48 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg shadow-lg py-1 z-50"
            >
              <button className="w-full px-4 py-2 text-left text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 flex items-center space-x-2">
                <User className="w-4 h-4" />
                <span>Profile</span>
              </button>
              <button className="w-full px-4 py-2 text-left text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 flex items-center space-x-2">
                <Settings className="w-4 h-4" />
                <span>Settings</span>
              </button>
              <div className="h-px bg-gray-200 dark:bg-gray-700 my-1" />
              <button className="w-full px-4 py-2 text-left text-sm text-red-600 dark:text-red-400 hover:bg-gray-100 dark:hover:bg-gray-700 flex items-center space-x-2">
                <LogOut className="w-4 h-4" />
                <span>Logout</span>
              </button>
            </motion.div>
          )}
        </div>
      </div>
    </motion.header>
  );
};