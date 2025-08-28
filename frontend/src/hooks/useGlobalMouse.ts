import React, { useState, useEffect, useCallback, useRef, useMemo } from 'react';

interface MousePosition {
  x: number;
  y: number;
}

interface MouseSubscriber {
  id: string;
  callback: (position: MousePosition) => void;
}

interface ClickSubscriber {
  id: string;
  callback: (event: MouseEvent) => void;
}

// Global mouse tracking singleton - very conservative implementation
class GlobalMouseTracker {
  private static instance: GlobalMouseTracker;
  private subscribers: Map<string, MouseSubscriber> = new Map();
  private clickSubscribers: Map<string, ClickSubscriber> = new Map();
  private currentPosition: MousePosition = { x: 0, y: 0 };
  private isListening = false;
  private isClickListening = false;

  static getInstance(): GlobalMouseTracker {
    if (!GlobalMouseTracker.instance) {
      GlobalMouseTracker.instance = new GlobalMouseTracker();
    }
    return GlobalMouseTracker.instance;
  }

  private handleMouseMove = (event: MouseEvent) => {
    this.currentPosition = { x: event.clientX, y: event.clientY };

    // Notify all subscribers
    this.subscribers.forEach(subscriber => {
      subscriber.callback(this.currentPosition);
    });
  };

  private handleClick = (event: MouseEvent) => {
    // Notify all click subscribers
    this.clickSubscribers.forEach(subscriber => {
      subscriber.callback(event);
    });
  };

  subscribe(id: string, callback: (position: MousePosition) => void): () => void {
    // Add subscriber
    this.subscribers.set(id, { id, callback });

    // Start listening if this is the first subscriber
    if (!this.isListening) {
      document.addEventListener('mousemove', this.handleMouseMove, { passive: true });
      this.isListening = true;
    }

    // Call callback immediately with current position
    callback(this.currentPosition);

    // Return unsubscribe function
    return () => {
      this.subscribers.delete(id);
      
      // Stop listening if no more subscribers
      if (this.subscribers.size === 0 && this.isListening) {
        document.removeEventListener('mousemove', this.handleMouseMove);
        this.isListening = false;
      }
    };
  }

  getCurrentPosition(): MousePosition {
    return this.currentPosition;
  }

  subscribeToClicks(id: string, callback: (event: MouseEvent) => void): () => void {
    // Add click subscriber
    this.clickSubscribers.set(id, { id, callback });

    // Start listening if this is the first click subscriber
    if (!this.isClickListening) {
      document.addEventListener('mousedown', this.handleClick, { passive: true });
      this.isClickListening = true;
    }

    // Return unsubscribe function
    return () => {
      this.clickSubscribers.delete(id);

      // Stop listening if no more click subscribers
      if (this.clickSubscribers.size === 0 && this.isClickListening) {
        document.removeEventListener('mousedown', this.handleClick);
        this.isClickListening = false;
      }
    };
  }

  getSubscriberCount(): number {
    return this.subscribers.size;
  }

  getClickSubscriberCount(): number {
    return this.clickSubscribers.size;
  }
}

// Basic hook for global mouse tracking - optimized implementation
export const useGlobalMouse = (
  callback?: (position: MousePosition) => void,
  dependencies: any[] = []
) => {
  const [mousePosition, setMousePosition] = useState<MousePosition>({ x: 0, y: 0 });
  const callbackRef = useRef(callback);

  // Update callback ref without causing re-renders
  useEffect(() => {
    callbackRef.current = callback;
  }, [callback]);

  const memoizedCallback = useCallback((position: MousePosition) => {
    setMousePosition(position);
    if (callbackRef.current) {
      callbackRef.current(position);
    }
  }, dependencies);

  useEffect(() => {
    const tracker = GlobalMouseTracker.getInstance();
    const subscriberId = `mouse-${Date.now()}-${Math.random()}`;

    const unsubscribe = tracker.subscribe(subscriberId, memoizedCallback);

    return unsubscribe;
  }, [memoizedCallback]);

  return mousePosition;
};

// Utility hook for specific mouse zones - optimized to prevent re-renders
export const useMouseZone = (
  zone: {
    left?: number;
    right?: number;
    top?: number;
    bottom?: number;
  },
  onEnter?: () => void,
  onLeave?: () => void
) => {
  const [isInZone, setIsInZone] = useState(false);
  const isInZoneRef = useRef(false);

  // Memoize the zone boundaries to prevent unnecessary re-renders
  const zoneBounds = useMemo(() => zone, [zone.left, zone.right, zone.top, zone.bottom]);

  // Memoize callbacks to prevent re-renders
  const stableOnEnter = useCallback(() => {
    if (onEnter) onEnter();
  }, [onEnter]);

  const stableOnLeave = useCallback(() => {
    if (onLeave) onLeave();
  }, [onLeave]);

  const handleMouseMove = useCallback((position: MousePosition) => {
    const inZone =
      (zoneBounds.left === undefined || position.x >= zoneBounds.left) &&
      (zoneBounds.right === undefined || position.x <= zoneBounds.right) &&
      (zoneBounds.top === undefined || position.y >= zoneBounds.top) &&
      (zoneBounds.bottom === undefined || position.y <= zoneBounds.bottom);

    if (inZone !== isInZoneRef.current) {
      isInZoneRef.current = inZone;
      setIsInZone(inZone);
      if (inZone) {
        stableOnEnter();
      } else {
        stableOnLeave();
      }
    }
  }, [zoneBounds, stableOnEnter, stableOnLeave]);

  useGlobalMouse(handleMouseMove, [handleMouseMove]);

  return isInZone;
};

// Hook for detecting clicks outside of elements - preserves exact behavior
export const useClickOutside = (
  refs: React.RefObject<HTMLElement | null> | React.RefObject<HTMLElement | null>[],
  callback: () => void,
  enabled: boolean = true
) => {
  useEffect(() => {
    if (!enabled) return;

    const tracker = GlobalMouseTracker.getInstance();
    const subscriberId = `click-outside-${Date.now()}-${Math.random()}`;

    const handleClick = (event: MouseEvent) => {
      const refArray = Array.isArray(refs) ? refs : [refs];
      const clickedOutside = refArray.every(ref =>
        ref.current && !ref.current.contains(event.target as Node)
      );

      if (clickedOutside) {
        callback();
      }
    };

    const unsubscribe = tracker.subscribeToClicks(subscriberId, handleClick);
    return unsubscribe;
  }, [refs, callback, enabled]);
};

export default useGlobalMouse;
