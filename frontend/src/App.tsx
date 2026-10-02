import React, { useState } from 'react';
import { DiscoverDevices, OpenInBrowser } from '../wailsjs/go/udpdiscovery/App';

interface Device {
  hostname: string;
  ip: string;
  port: string;
  url: string;
}

export default function App() {
  const [devices, setDevices] = useState<Device[]>([]);
  const [scanning, setScanning] = useState(false);

  const handleScan = async () => {
    setScanning(true);
    try {
      const result = await DiscoverDevices();
      setDevices(result || []);
    } catch (err) {
      console.error('Discovery error:', err);
    } finally {
      setScanning(false);
    }
  };

  return (
    <div style={{ padding: '2rem', fontFamily: 'system-ui, sans-serif', color: '#1e293b' }}>
      <h2>IBS-Tool LAN Discovery</h2>
      <button
        onClick={handleScan}
        disabled={scanning}
        style={{
          padding: '0.6rem 1.2rem',
          fontSize: '1rem',
          backgroundColor: scanning ? '#94a3b8' : '#2563eb',
          color: '#ffffff',
          border: 'none',
          borderRadius: '6px',
          cursor: scanning ? 'not-allowed' : 'pointer',
        }}
      >
        {scanning ? 'Searching LAN...' : 'Search for IBS-Tool'}
      </button>

      <div style={{ marginTop: '1.5rem' }}>
        {devices.length === 0 && !scanning && (
          <p style={{ color: '#64748b' }}>No devices found yet. Click Search to broadcast.</p>
        )}

        {devices.map((dev, idx) => (
          <div
            key={idx}
            onClick={() => OpenInBrowser(dev.url)}
            style={{
              display: 'flex',
              justifyContent: 'space-between',
              alignItems: 'center',
              padding: '1rem',
              marginBottom: '0.75rem',
              border: '1px solid #e2e8f0',
              borderRadius: '8px',
              backgroundColor: '#f8fafc',
              cursor: 'pointer',
              transition: 'border-color 0.2s',
            }}
            onMouseEnter={(e) => (e.currentTarget.style.borderColor = '#2563eb')}
            onMouseLeave={(e) => (e.currentTarget.style.borderColor = '#e2e8f0')}
          >
            <div>
              <strong style={{ fontSize: '1.1rem', display: 'block' }}>
                {dev.hostname || 'ibsTool Node'}
              </strong>
              <span style={{ color: '#64748b', fontSize: '0.9rem' }}>{dev.ip}{dev.port}</span>
            </div>
            <span style={{ color: '#2563eb', fontWeight: 600 }}>Open in Browser →</span>
          </div>
        ))}
      </div>
    </div>
  );
}