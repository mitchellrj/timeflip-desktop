import React, { useEffect, useMemo, useState } from 'react';
import { createRoot } from 'react-dom/client';
import { Bluetooth, Clock3, History, Pause, Play, RefreshCw, Settings, SlidersHorizontal, Tag } from 'lucide-react';
import { Events } from '@wailsio/runtime';
import { GetAppState, ScanDevices, SetPaused } from '../bindings/github.com/mitchellrj/timeflip-desktop/internal/app/controller.js';
import './styles.css';

function App() {
  const [state, setState] = useState({ devices: [], states: [], tasks: [], sessions: [], facetConfigs: [] });
  const [error, setError] = useState('');
  const [selectedDevice, setSelectedDevice] = useState('');

  async function refresh() {
    try {
      const next = await GetAppState();
      setState(next);
      if (!selectedDevice && next.devices?.length) {
        setSelectedDevice(next.devices[0].id);
      }
      setError('');
    } catch (err) {
      setError(String(err));
    }
  }

  useEffect(() => {
    refresh();
    const offRefresh = Events.On('shell.refresh', refresh);
    return () => {
      offRefresh();
    };
  }, []);

  const activeState = useMemo(() => state.states?.find((item) => item.deviceID === selectedDevice) || state.states?.[0], [state, selectedDevice]);
  const currentSession = state.currentSession;

  return (
    <main className="app">
      <aside className="sidebar">
        <div className="brand">
          <div className="mark">T</div>
          <div>
            <h1>TimeFlip</h1>
            <p>Local desktop tracking</p>
          </div>
        </div>
        <button className="primary" onClick={refresh}><RefreshCw size={16} /> Refresh</button>
        <nav>
          <a href="#dashboard"><Clock3 size={17} /> Dashboard</a>
          <a href="#devices"><Bluetooth size={17} /> Devices</a>
          <a href="#facets"><SlidersHorizontal size={17} /> Facets</a>
          <a href="#history"><History size={17} /> History</a>
          <a href="#settings"><Settings size={17} /> Settings</a>
        </nav>
      </aside>

      <section className="content">
        {error && <div className="notice">{error}</div>}
        <section id="dashboard" className="band dashboard">
          <div>
            <span className="eyebrow">Current state</span>
            <h2>{currentSession?.taskLabelSnapshot || (activeState?.paused ? 'Paused' : 'No active session')}</h2>
            <p>{activeState?.connectionState || 'No device connected'} · facet {activeState?.currentFacetKnown ? activeState.currentFacet : 'unknown'}</p>
          </div>
            <button className="iconButton" onClick={() => selectedDevice && SetPaused(selectedDevice, !activeState?.paused).then(refresh)}>
            {activeState?.paused ? <Play size={20} /> : <Pause size={20} />}
          </button>
        </section>

        <section id="devices" className="grid two">
          <Panel title="Devices" icon={<Bluetooth size={18} />}>
            <button onClick={() => ScanDevices().then(refresh)}><RefreshCw size={16} /> Scan</button>
            <div className="list">
              {state.devices?.map((device) => (
                <button key={device.id} className={`row ${selectedDevice === device.id ? 'selected' : ''}`} onClick={() => setSelectedDevice(device.id)}>
                  <span>{device.displayName || device.id}</span>
                  <small>{device.pairingState || 'known'}</small>
                </button>
              ))}
            </div>
          </Panel>
          <Panel title="Status" icon={<Tag size={18} />}>
            <dl className="facts">
              <dt>Connection</dt><dd>{activeState?.connectionState || 'none'}</dd>
              <dt>Battery</dt><dd>{activeState?.batteryPercent ? `${activeState.batteryPercent}%` : 'unknown'}</dd>
              <dt>Locked</dt><dd>{activeState?.locked ? 'yes' : 'no'}</dd>
              <dt>Paused</dt><dd>{activeState?.paused ? 'yes' : 'no'}</dd>
            </dl>
          </Panel>
        </section>

        <section id="facets" className="band">
          <div className="sectionTitle">
            <h2>Facets</h2>
            <p>Labels and icons stay local. Colours can be confirmed on device when connected.</p>
          </div>
          <div className="facetGrid">
            {Array.from({ length: 12 }, (_, index) => {
              const facet = index + 1;
              const cfg = state.facetConfigs?.find((item) => item.facet === facet);
              return (
                <div className="facet" key={facet}>
                  <span className="swatch" style={{ background: cfg?.color || '#d8dee9' }} />
                  <strong>{cfg?.label || `Facet ${facet}`}</strong>
                  <small>{cfg?.isPauseAssignment ? 'Pause side' : cfg?.icon || 'Unassigned'}</small>
                </div>
              );
            })}
          </div>
        </section>

        <section id="history" className="band">
          <div className="sectionTitle">
            <h2>Task Sessions</h2>
            <p>Summary reporting comes later; this view starts with reliable sessions.</p>
          </div>
          <div className="sessionList">
            {state.sessions?.slice(0, 12).map((session) => (
              <div className="session" key={session.id}>
                <span className="swatch" style={{ background: session.taskColorSnapshot || '#d8dee9' }} />
                <strong>{session.taskLabelSnapshot}</strong>
                <small>Facet {session.facet} · {Math.round((session.durationSeconds || 0) / 60)} min</small>
              </div>
            ))}
          </div>
        </section>
      </section>
    </main>
  );
}

function Panel({ title, icon, children }) {
  return (
    <section className="panel">
      <h2>{icon}{title}</h2>
      {children}
    </section>
  );
}

createRoot(document.getElementById('root')).render(<App />);
