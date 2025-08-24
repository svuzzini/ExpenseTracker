// Service Worker for ExpenseTracker PWA

const CACHE_NAME = 'expense-tracker-v1.0.0';
const STATIC_CACHE_NAME = 'expense-tracker-static-v1.0.0';
const DYNAMIC_CACHE_NAME = 'expense-tracker-dynamic-v1.0.0';

// Files to cache for offline functionality
const STATIC_ASSETS = [
  '/',
  '/dashboard',
  '/static/css/custom.css',
  '/static/js/utils.js',
  '/static/js/auth.js',
  '/static/js/websocket.js',
  '/static/js/app.js',
  '/static/manifest.json',
  // Add core images and icons here
  '/static/images/icon-192x192.png',
  '/static/images/icon-512x512.png',
  // CDN resources for offline fallback
  'https://cdn.tailwindcss.com',
  'https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.4.0/css/all.min.css'
];

// API endpoints to cache for offline access
const API_CACHE_PATTERNS = [
  /^\/api\/v1\/events\/$/,
  /^\/api\/v1\/events\/\d+$/,
  /^\/api\/v1\/categories$/,
  /^\/api\/v1\/auth\/profile$/
];

// Install event - cache static assets
self.addEventListener('install', (event) => {
  console.log('Service Worker: Installing...');
  
  event.waitUntil(
    Promise.all([
      // Cache static assets
      caches.open(STATIC_CACHE_NAME).then((cache) => {
        console.log('Service Worker: Caching static assets');
        return cache.addAll(STATIC_ASSETS.map(url => new Request(url, { credentials: 'same-origin' })));
      }),
      // Skip waiting to activate immediately
      self.skipWaiting()
    ])
  );
});

// Activate event - clean up old caches
self.addEventListener('activate', (event) => {
  console.log('Service Worker: Activating...');
  
  event.waitUntil(
    Promise.all([
      // Clean up old caches
      caches.keys().then((cacheNames) => {
        return Promise.all(
          cacheNames.map((cacheName) => {
            if (cacheName !== STATIC_CACHE_NAME && 
                cacheName !== DYNAMIC_CACHE_NAME &&
                cacheName.startsWith('expense-tracker-')) {
              console.log('Service Worker: Deleting old cache', cacheName);
              return caches.delete(cacheName);
            }
          })
        );
      }),
      // Take control of all clients
      self.clients.claim()
    ])
  );
});

// Fetch event - handle requests with caching strategies
self.addEventListener('fetch', (event) => {
  const { request } = event;
  const url = new URL(request.url);
  
  // Skip non-GET requests
  if (request.method !== 'GET') {
    return;
  }
  
  // Skip Chrome extensions and other protocols
  if (!url.protocol.startsWith('http')) {
    return;
  }
  
  // Handle different types of requests
  if (isStaticAsset(request)) {
    event.respondWith(handleStaticAsset(request));
  } else if (isAPIRequest(request)) {
    event.respondWith(handleAPIRequest(request));
  } else if (isImageRequest(request)) {
    event.respondWith(handleImageRequest(request));
  } else if (isNavigationRequest(request)) {
    event.respondWith(handleNavigationRequest(request));
  } else {
    event.respondWith(handleOtherRequest(request));
  }
});

// Check if request is for static assets
function isStaticAsset(request) {
  const url = new URL(request.url);
  return url.pathname.startsWith('/static/') ||
         STATIC_ASSETS.some(asset => url.pathname === asset || url.href === asset);
}

// Check if request is for API
function isAPIRequest(request) {
  const url = new URL(request.url);
  return url.pathname.startsWith('/api/');
}

// Check if request is for images
function isImageRequest(request) {
  return request.destination === 'image' ||
         /\.(jpg|jpeg|png|gif|webp|svg|ico)$/i.test(new URL(request.url).pathname);
}

// Check if request is navigation
function isNavigationRequest(request) {
  return request.mode === 'navigate';
}

// Handle static assets - Cache First strategy
async function handleStaticAsset(request) {
  try {
    const cachedResponse = await caches.match(request);
    if (cachedResponse) {
      return cachedResponse;
    }
    
    const networkResponse = await fetch(request);
    if (networkResponse.ok) {
      const cache = await caches.open(STATIC_CACHE_NAME);
      cache.put(request, networkResponse.clone());
    }
    return networkResponse;
  } catch (error) {
    console.log('Service Worker: Static asset fetch failed', error);
    return new Response('Offline', { status: 503 });
  }
}

// Handle API requests - Network First with cache fallback
async function handleAPIRequest(request) {
  try {
    const networkResponse = await fetch(request);
    
    if (networkResponse.ok) {
      // Cache successful responses for specific endpoints
      if (shouldCacheAPIResponse(request)) {
        const cache = await caches.open(DYNAMIC_CACHE_NAME);
        cache.put(request, networkResponse.clone());
      }
    }
    
    return networkResponse;
  } catch (error) {
    console.log('Service Worker: API request failed, trying cache', error);
    
    const cachedResponse = await caches.match(request);
    if (cachedResponse) {
      // Add offline indicator header
      const response = cachedResponse.clone();
      response.headers.set('X-Served-By', 'ServiceWorker-Cache');
      return response;
    }
    
    // Return offline response for critical API endpoints
    if (isCriticalAPIEndpoint(request)) {
      return createOfflineAPIResponse(request);
    }
    
    throw error;
  }
}

// Handle image requests - Cache First with network fallback
async function handleImageRequest(request) {
  try {
    const cachedResponse = await caches.match(request);
    if (cachedResponse) {
      return cachedResponse;
    }
    
    const networkResponse = await fetch(request);
    if (networkResponse.ok) {
      const cache = await caches.open(DYNAMIC_CACHE_NAME);
      cache.put(request, networkResponse.clone());
    }
    return networkResponse;
  } catch (error) {
    console.log('Service Worker: Image fetch failed', error);
    // Return placeholder image for failed image loads
    return createPlaceholderImage();
  }
}

// Handle navigation requests - Network First with offline fallback
async function handleNavigationRequest(request) {
  try {
    const networkResponse = await fetch(request);
    return networkResponse;
  } catch (error) {
    console.log('Service Worker: Navigation failed, serving offline page');
    
    // Serve cached dashboard or offline page
    const cachedDashboard = await caches.match('/dashboard');
    if (cachedDashboard) {
      return cachedDashboard;
    }
    
    return createOfflinePage();
  }
}

// Handle other requests - Network with cache fallback
async function handleOtherRequest(request) {
  try {
    const networkResponse = await fetch(request);
    
    if (networkResponse.ok) {
      const cache = await caches.open(DYNAMIC_CACHE_NAME);
      cache.put(request, networkResponse.clone());
    }
    
    return networkResponse;
  } catch (error) {
    const cachedResponse = await caches.match(request);
    if (cachedResponse) {
      return cachedResponse;
    }
    
    throw error;
  }
}

// Check if API response should be cached
function shouldCacheAPIResponse(request) {
  const url = new URL(request.url);
  return API_CACHE_PATTERNS.some(pattern => pattern.test(url.pathname));
}

// Check if API endpoint is critical for offline functionality
function isCriticalAPIEndpoint(request) {
  const url = new URL(request.url);
  return url.pathname === '/api/v1/events/' ||
         url.pathname === '/api/v1/auth/profile' ||
         url.pathname === '/api/v1/categories';
}

// Create offline API response
function createOfflineAPIResponse(request) {
  const url = new URL(request.url);
  
  let offlineData = {};
  
  if (url.pathname === '/api/v1/events/') {
    offlineData = [];
  } else if (url.pathname === '/api/v1/categories') {
    offlineData = [
      { id: 1, name: 'Food & Dining', icon: '🍽️' },
      { id: 2, name: 'Transportation', icon: '🚗' },
      { id: 3, name: 'Accommodation', icon: '🏨' },
      { id: 4, name: 'Other', icon: '📦' }
    ];
  }
  
  return new Response(JSON.stringify(offlineData), {
    status: 200,
    statusText: 'OK (Offline)',
    headers: {
      'Content-Type': 'application/json',
      'X-Served-By': 'ServiceWorker-Offline'
    }
  });
}

// Create placeholder image response
function createPlaceholderImage() {
  // Simple 1x1 transparent PNG
  const imageData = 'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg==';
  const imageBytes = Uint8Array.from(atob(imageData), c => c.charCodeAt(0));
  
  return new Response(imageBytes, {
    status: 200,
    statusText: 'OK (Placeholder)',
    headers: {
      'Content-Type': 'image/png',
      'Cache-Control': 'no-cache'
    }
  });
}

// Create offline page response
function createOfflinePage() {
  const offlineHTML = `
    <!DOCTYPE html>
    <html lang="en">
    <head>
        <meta charset="UTF-8">
        <meta name="viewport" content="width=device-width, initial-scale=1.0">
        <title>ExpenseTracker - Offline</title>
        <script src="https://cdn.tailwindcss.com"></script>
        <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.4.0/css/all.min.css">
    </head>
    <body class="bg-gray-50 min-h-screen flex items-center justify-center px-4">
        <div class="text-center max-w-md">
            <div class="mb-6">
                <i class="fas fa-wifi-slash text-6xl text-gray-400 mb-4"></i>
                <h1 class="text-2xl font-bold text-gray-900 mb-2">You're Offline</h1>
                <p class="text-gray-600">
                    No internet connection available. Please check your connection and try again.
                </p>
            </div>
            
            <div class="space-y-4">
                <button onclick="window.location.reload()" class="w-full bg-blue-600 text-white py-2 px-4 rounded-lg hover:bg-blue-700 transition-colors">
                    <i class="fas fa-redo mr-2"></i>Try Again
                </button>
                
                <button onclick="goToLastCachedPage()" class="w-full border border-gray-300 text-gray-700 py-2 px-4 rounded-lg hover:bg-gray-50 transition-colors">
                    <i class="fas fa-home mr-2"></i>Go to Dashboard
                </button>
            </div>
            
            <div class="mt-8 text-sm text-gray-500">
                <p>Some features may be limited while offline.</p>
            </div>
        </div>
        
        <script>
            function goToLastCachedPage() {
                window.location.href = '/dashboard';
            }
            
            // Auto-retry when online
            window.addEventListener('online', () => {
                window.location.reload();
            });
        </script>
    </body>
    </html>
  `;
  
  return new Response(offlineHTML, {
    status: 200,
    statusText: 'OK (Offline)',
    headers: {
      'Content-Type': 'text/html',
      'Cache-Control': 'no-cache'
    }
  });
}

// Background sync for offline actions
self.addEventListener('sync', (event) => {
  if (event.tag === 'background-sync-expenses') {
    event.waitUntil(syncOfflineExpenses());
  } else if (event.tag === 'background-sync-contributions') {
    event.waitUntil(syncOfflineContributions());
  }
});

// Sync offline expenses when back online
async function syncOfflineExpenses() {
  console.log('Service Worker: Syncing offline expenses...');
  
  try {
    const offlineExpenses = await getOfflineData('expenses');
    
    for (const expense of offlineExpenses) {
      try {
        const response = await fetch('/api/v1/expenses/', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${expense.token}`
          },
          body: JSON.stringify(expense.data)
        });
        
        if (response.ok) {
          await removeOfflineData('expenses', expense.id);
          console.log('Service Worker: Synced offline expense', expense.id);
        }
      } catch (error) {
        console.log('Service Worker: Failed to sync expense', expense.id, error);
      }
    }
  } catch (error) {
    console.log('Service Worker: Background sync failed', error);
  }
}

// Sync offline contributions when back online
async function syncOfflineContributions() {
  console.log('Service Worker: Syncing offline contributions...');
  
  try {
    const offlineContributions = await getOfflineData('contributions');
    
    for (const contribution of offlineContributions) {
      try {
        const response = await fetch('/api/v1/contributions/', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${contribution.token}`
          },
          body: JSON.stringify(contribution.data)
        });
        
        if (response.ok) {
          await removeOfflineData('contributions', contribution.id);
          console.log('Service Worker: Synced offline contribution', contribution.id);
        }
      } catch (error) {
        console.log('Service Worker: Failed to sync contribution', contribution.id, error);
      }
    }
  } catch (error) {
    console.log('Service Worker: Background sync failed', error);
  }
}

// Get offline data from IndexedDB
async function getOfflineData(type) {
  return new Promise((resolve, reject) => {
    const request = indexedDB.open('ExpenseTrackerOffline', 1);
    
    request.onerror = () => reject(request.error);
    request.onsuccess = () => {
      const db = request.result;
      const transaction = db.transaction([type], 'readonly');
      const store = transaction.objectStore(type);
      const getAllRequest = store.getAll();
      
      getAllRequest.onsuccess = () => resolve(getAllRequest.result);
      getAllRequest.onerror = () => reject(getAllRequest.error);
    };
    
    request.onupgradeneeded = () => {
      const db = request.result;
      if (!db.objectStoreNames.contains(type)) {
        db.createObjectStore(type, { keyPath: 'id' });
      }
    };
  });
}

// Remove offline data from IndexedDB
async function removeOfflineData(type, id) {
  return new Promise((resolve, reject) => {
    const request = indexedDB.open('ExpenseTrackerOffline', 1);
    
    request.onerror = () => reject(request.error);
    request.onsuccess = () => {
      const db = request.result;
      const transaction = db.transaction([type], 'readwrite');
      const store = transaction.objectStore(type);
      const deleteRequest = store.delete(id);
      
      deleteRequest.onsuccess = () => resolve();
      deleteRequest.onerror = () => reject(deleteRequest.error);
    };
  });
}

// Handle push notifications
self.addEventListener('push', (event) => {
  if (!event.data) return;
  
  const data = event.data.json();
  const options = {
    body: data.body || 'New expense tracker notification',
    icon: '/static/images/icon-192x192.png',
    badge: '/static/images/badge-72x72.png',
    image: data.image,
    data: data.data,
    actions: data.actions || [
      {
        action: 'view',
        title: 'View',
        icon: '/static/images/action-view.png'
      },
      {
        action: 'dismiss',
        title: 'Dismiss',
        icon: '/static/images/action-dismiss.png'
      }
    ],
    tag: data.tag || 'general',
    renotify: true,
    requireInteraction: data.requireInteraction || false,
    silent: false,
    vibrate: [200, 100, 200]
  };
  
  event.waitUntil(
    self.registration.showNotification(data.title || 'ExpenseTracker', options)
  );
});

// Handle notification clicks
self.addEventListener('notificationclick', (event) => {
  event.notification.close();
  
  const action = event.action;
  const data = event.notification.data;
  
  if (action === 'view' || !action) {
    // Open the app or navigate to specific page
    const urlToOpen = data && data.url ? data.url : '/dashboard';
    
    event.waitUntil(
      clients.matchAll({ type: 'window', includeUncontrolled: true }).then((clientList) => {
        // Check if app is already open
        for (const client of clientList) {
          if (client.url.includes(urlToOpen) && 'focus' in client) {
            return client.focus();
          }
        }
        
        // Open new window/tab
        if (clients.openWindow) {
          return clients.openWindow(urlToOpen);
        }
      })
    );
  }
  // Handle other actions (dismiss is handled by closing notification)
});

// Handle message events from the main app
self.addEventListener('message', (event) => {
  const { type, data } = event.data;
  
  switch (type) {
    case 'SKIP_WAITING':
      self.skipWaiting();
      break;
      
    case 'GET_VERSION':
      event.ports[0].postMessage({ version: CACHE_NAME });
      break;
      
    case 'CACHE_URLS':
      event.waitUntil(
        caches.open(DYNAMIC_CACHE_NAME).then((cache) => {
          return cache.addAll(data.urls);
        })
      );
      break;
      
    case 'CLEAR_CACHE':
      event.waitUntil(
        caches.delete(data.cacheName || DYNAMIC_CACHE_NAME)
      );
      break;
  }
});

console.log('Service Worker: Script loaded', CACHE_NAME);
