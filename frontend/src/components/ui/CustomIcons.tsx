import React from 'react';

interface IconProps {
  className?: string;
  size?: number;
}

// Client Icons
export const ClientIcon: React.FC<IconProps> = ({ className = "", size = 24 }) => (
  <svg width={size} height={size} viewBox="0 0 24 24" fill="none" className={className}>
    <rect x="2" y="3" width="20" height="14" rx="2" stroke="currentColor" strokeWidth="2" fill="none"/>
    <path d="M8 21h8" stroke="currentColor" strokeWidth="2"/>
    <path d="M12 17v4" stroke="currentColor" strokeWidth="2"/>
    <circle cx="12" cy="10" r="3" fill="currentColor" opacity="0.3"/>
    <path d="M9 13h6" stroke="currentColor" strokeWidth="1.5"/>
  </svg>
);

export const MobileIcon: React.FC<IconProps> = ({ className = "", size = 24 }) => (
  <svg width={size} height={size} viewBox="0 0 24 24" fill="none" className={className}>
    <rect x="5" y="2" width="14" height="20" rx="2" stroke="currentColor" strokeWidth="2" fill="none"/>
    <path d="M12 18h.01" stroke="currentColor" strokeWidth="2" strokeLinecap="round"/>
    <rect x="7" y="5" width="10" height="8" rx="1" fill="currentColor" opacity="0.2"/>
    <path d="M9 7h6" stroke="currentColor" strokeWidth="1"/>
    <path d="M9 9h4" stroke="currentColor" strokeWidth="1"/>
  </svg>
);

// Compute Icons
export const ServerIcon: React.FC<IconProps> = ({ className = "", size = 24 }) => (
  <svg width={size} height={size} viewBox="0 0 24 24" fill="none" className={className}>
    <rect x="2" y="3" width="20" height="4" rx="1" stroke="currentColor" strokeWidth="2" fill="currentColor" fillOpacity="0.1"/>
    <rect x="2" y="10" width="20" height="4" rx="1" stroke="currentColor" strokeWidth="2" fill="currentColor" fillOpacity="0.1"/>
    <rect x="2" y="17" width="20" height="4" rx="1" stroke="currentColor" strokeWidth="2" fill="currentColor" fillOpacity="0.1"/>
    <circle cx="6" cy="5" r="1" fill="currentColor"/>
    <circle cx="6" cy="12" r="1" fill="currentColor"/>
    <circle cx="6" cy="19" r="1" fill="currentColor"/>
    <path d="M10 5h8" stroke="currentColor" strokeWidth="1"/>
    <path d="M10 12h6" stroke="currentColor" strokeWidth="1"/>
    <path d="M10 19h4" stroke="currentColor" strokeWidth="1"/>
  </svg>
);

export const MicroserviceIcon: React.FC<IconProps> = ({ className = "", size = 24 }) => (
  <svg width={size} height={size} viewBox="0 0 24 24" fill="none" className={className}>
    <rect x="3" y="3" width="6" height="6" rx="1" stroke="currentColor" strokeWidth="2" fill="currentColor" fillOpacity="0.2"/>
    <rect x="15" y="3" width="6" height="6" rx="1" stroke="currentColor" strokeWidth="2" fill="currentColor" fillOpacity="0.2"/>
    <rect x="3" y="15" width="6" height="6" rx="1" stroke="currentColor" strokeWidth="2" fill="currentColor" fillOpacity="0.2"/>
    <rect x="15" y="15" width="6" height="6" rx="1" stroke="currentColor" strokeWidth="2" fill="currentColor" fillOpacity="0.2"/>
    <path d="M9 6h6" stroke="currentColor" strokeWidth="2"/>
    <path d="M6 9v6" stroke="currentColor" strokeWidth="2"/>
    <path d="M18 9v6" stroke="currentColor" strokeWidth="2"/>
    <path d="M9 18h6" stroke="currentColor" strokeWidth="2"/>
    <circle cx="12" cy="12" r="2" fill="currentColor"/>
  </svg>
);

export const LoadBalancerIcon: React.FC<IconProps> = ({ className = "", size = 24 }) => (
  <svg width={size} height={size} viewBox="0 0 24 24" fill="none" className={className}>
    <circle cx="12" cy="12" r="8" stroke="currentColor" strokeWidth="2" fill="none"/>
    <circle cx="12" cy="12" r="3" fill="currentColor" opacity="0.3"/>
    <path d="M12 4v4" stroke="currentColor" strokeWidth="2"/>
    <path d="M12 16v4" stroke="currentColor" strokeWidth="2"/>
    <path d="M4 12h4" stroke="currentColor" strokeWidth="2"/>
    <path d="M16 12h4" stroke="currentColor" strokeWidth="2"/>
    <path d="M7.5 7.5l2.8 2.8" stroke="currentColor" strokeWidth="2"/>
    <path d="M13.7 13.7l2.8 2.8" stroke="currentColor" strokeWidth="2"/>
    <path d="M7.5 16.5l2.8-2.8" stroke="currentColor" strokeWidth="2"/>
    <path d="M13.7 10.3l2.8-2.8" stroke="currentColor" strokeWidth="2"/>
  </svg>
);

// Storage Icons
export const DatabaseIcon: React.FC<IconProps> = ({ className = "", size = 24 }) => (
  <svg width={size} height={size} viewBox="0 0 24 24" fill="none" className={className}>
    <ellipse cx="12" cy="5" rx="9" ry="3" stroke="currentColor" strokeWidth="2" fill="currentColor" fillOpacity="0.2"/>
    <path d="M21 12c0 1.66-4 3-9 3s-9-1.34-9-3" stroke="currentColor" strokeWidth="2"/>
    <path d="M3 5v14c0 1.66 4 3 9 3s9-1.34 9-3V5" stroke="currentColor" strokeWidth="2" fill="none"/>
    <ellipse cx="12" cy="12" rx="7" ry="2" fill="currentColor" opacity="0.1"/>
    <ellipse cx="12" cy="19" rx="7" ry="2" fill="currentColor" opacity="0.1"/>
  </svg>
);

export const CacheIcon: React.FC<IconProps> = ({ className = "", size = 24 }) => (
  <svg width={size} height={size} viewBox="0 0 24 24" fill="none" className={className}>
    <rect x="3" y="6" width="18" height="12" rx="2" stroke="currentColor" strokeWidth="2" fill="currentColor" fillOpacity="0.1"/>
    <path d="M7 10h10" stroke="currentColor" strokeWidth="2"/>
    <path d="M7 14h6" stroke="currentColor" strokeWidth="2"/>
    <circle cx="18" cy="8" r="2" fill="currentColor"/>
    <path d="M16 8l1 1 2-2" stroke="white" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round"/>
    <rect x="5" y="8" width="2" height="8" rx="1" fill="currentColor" opacity="0.3"/>
  </svg>
);

export const ObjectStorageIcon: React.FC<IconProps> = ({ className = "", size = 24 }) => (
  <svg width={size} height={size} viewBox="0 0 24 24" fill="none" className={className}>
    <rect x="3" y="4" width="18" height="16" rx="2" stroke="currentColor" strokeWidth="2" fill="none"/>
    <rect x="6" y="7" width="3" height="3" rx="0.5" fill="currentColor" opacity="0.4"/>
    <rect x="10" y="7" width="3" height="3" rx="0.5" fill="currentColor" opacity="0.4"/>
    <rect x="14" y="7" width="3" height="3" rx="0.5" fill="currentColor" opacity="0.4"/>
    <rect x="6" y="11" width="3" height="3" rx="0.5" fill="currentColor" opacity="0.4"/>
    <rect x="10" y="11" width="3" height="3" rx="0.5" fill="currentColor" opacity="0.4"/>
    <rect x="14" y="11" width="3" height="3" rx="0.5" fill="currentColor" opacity="0.4"/>
    <rect x="6" y="15" width="3" height="3" rx="0.5" fill="currentColor" opacity="0.4"/>
    <rect x="10" y="15" width="3" height="3" rx="0.5" fill="currentColor" opacity="0.4"/>
    <rect x="14" y="15" width="3" height="3" rx="0.5" fill="currentColor" opacity="0.4"/>
  </svg>
);

// Network Icons
export const CDNIcon: React.FC<IconProps> = ({ className = "", size = 24 }) => (
  <svg width={size} height={size} viewBox="0 0 24 24" fill="none" className={className}>
    <circle cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="2" fill="none"/>
    <circle cx="12" cy="12" r="6" stroke="currentColor" strokeWidth="1.5" fill="currentColor" fillOpacity="0.1"/>
    <circle cx="12" cy="12" r="2" fill="currentColor"/>
    <path d="M12 2v4" stroke="currentColor" strokeWidth="2"/>
    <path d="M12 18v4" stroke="currentColor" strokeWidth="2"/>
    <path d="M2 12h4" stroke="currentColor" strokeWidth="2"/>
    <path d="M18 12h4" stroke="currentColor" strokeWidth="2"/>
    <path d="M5.6 5.6l2.8 2.8" stroke="currentColor" strokeWidth="1.5"/>
    <path d="M15.6 15.6l2.8 2.8" stroke="currentColor" strokeWidth="1.5"/>
    <path d="M5.6 18.4l2.8-2.8" stroke="currentColor" strokeWidth="1.5"/>
    <path d="M15.6 8.4l2.8-2.8" stroke="currentColor" strokeWidth="1.5"/>
  </svg>
);

export const APIGatewayIcon: React.FC<IconProps> = ({ className = "", size = 24 }) => (
  <svg width={size} height={size} viewBox="0 0 24 24" fill="none" className={className}>
    <rect x="2" y="8" width="20" height="8" rx="2" stroke="currentColor" strokeWidth="2" fill="currentColor" fillOpacity="0.1"/>
    <path d="M6 4v4" stroke="currentColor" strokeWidth="2"/>
    <path d="M12 4v4" stroke="currentColor" strokeWidth="2"/>
    <path d="M18 4v4" stroke="currentColor" strokeWidth="2"/>
    <path d="M6 16v4" stroke="currentColor" strokeWidth="2"/>
    <path d="M12 16v4" stroke="currentColor" strokeWidth="2"/>
    <path d="M18 16v4" stroke="currentColor" strokeWidth="2"/>
    <circle cx="8" cy="12" r="1.5" fill="currentColor"/>
    <circle cx="12" cy="12" r="1.5" fill="currentColor"/>
    <circle cx="16" cy="12" r="1.5" fill="currentColor"/>
    <rect x="10" y="10" width="4" height="4" rx="1" stroke="currentColor" strokeWidth="1" fill="none"/>
  </svg>
);

// Processing Icons
export const MessageQueueIcon: React.FC<IconProps> = ({ className = "", size = 24 }) => (
  <svg width={size} height={size} viewBox="0 0 24 24" fill="none" className={className}>
    <rect x="2" y="6" width="20" height="12" rx="2" stroke="currentColor" strokeWidth="2" fill="none"/>
    <rect x="4" y="8" width="3" height="8" rx="0.5" fill="currentColor" opacity="0.3"/>
    <rect x="8" y="8" width="3" height="8" rx="0.5" fill="currentColor" opacity="0.5"/>
    <rect x="12" y="8" width="3" height="8" rx="0.5" fill="currentColor" opacity="0.7"/>
    <rect x="16" y="8" width="3" height="8" rx="0.5" fill="currentColor" opacity="0.4"/>
    <path d="M20 10l2 2-2 2" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
    <circle cx="6" cy="4" r="1" fill="currentColor"/>
    <circle cx="10" cy="4" r="1" fill="currentColor"/>
    <circle cx="14" cy="4" r="1" fill="currentColor"/>
  </svg>
);

// Security Icons
export const FirewallIcon: React.FC<IconProps> = ({ className = "", size = 24 }) => (
  <svg width={size} height={size} viewBox="0 0 24 24" fill="none" className={className}>
    <path d="M12 2l8 3v7c0 5.55-3.84 10.74-9 12-5.16-1.26-9-6.45-9-12V5l8-3z" stroke="currentColor" strokeWidth="2" fill="currentColor" fillOpacity="0.1"/>
    <path d="M8 12l2 2 4-4" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
    <rect x="10" y="6" width="4" height="2" rx="1" fill="currentColor" opacity="0.5"/>
    <rect x="9" y="16" width="6" height="1" rx="0.5" fill="currentColor" opacity="0.3"/>
  </svg>
);

// Monitoring Icons
export const MonitoringIcon: React.FC<IconProps> = ({ className = "", size = 24 }) => (
  <svg width={size} height={size} viewBox="0 0 24 24" fill="none" className={className}>
    <rect x="2" y="3" width="20" height="14" rx="2" stroke="currentColor" strokeWidth="2" fill="none"/>
    <path d="M8 21h8" stroke="currentColor" strokeWidth="2"/>
    <path d="M12 17v4" stroke="currentColor" strokeWidth="2"/>
    <path d="M6 8l2 2 2-2 2 2 2-2 2 2" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" fill="none"/>
    <rect x="4" y="11" width="16" height="4" rx="1" fill="currentColor" opacity="0.2"/>
    <circle cx="18" cy="6" r="2" fill="currentColor"/>
  </svg>
);

// Get icon by component type
export const getComponentIcon = (type: string, props: IconProps = {}) => {
  const iconMap: Record<string, React.FC<IconProps>> = {
    client: ClientIcon,
    mobile: MobileIcon,
    webserver: ServerIcon,
    microservice: MicroserviceIcon,
    loadbalancer: LoadBalancerIcon,
    database: DatabaseIcon,
    cache: CacheIcon,
    storage: ObjectStorageIcon,
    cdn: CDNIcon,
    gateway: APIGatewayIcon,
    queue: MessageQueueIcon,
    security: FirewallIcon,
    monitoring: MonitoringIcon,
  };

  const IconComponent = iconMap[type] || ServerIcon;
  return <IconComponent {...props} />;
};