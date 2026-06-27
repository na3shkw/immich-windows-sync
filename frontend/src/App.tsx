import { useState, useEffect } from 'react';
import { LoadConfig, SaveConfig, SelectFolder, StartWatcher, StopWatcher, SyncNow, IsStartupRegistered, RegisterStartup, UnRegisterStartup } from '../wailsjs/go/main/App';
import { config } from '../wailsjs/go/models';

type Page = 'connection' | 'folders' | 'status' | 'startup';
type SaveStatus = 'idle' | 'saved' | 'error';
type WatcherStatus = 'stopped' | 'running';

function App() {
  const [activePage, setActivePage] = useState<Page>('connection');
  const [cfg, setCfg] = useState<config.Config>(
    config.Config.createFrom({ immich: { serverURL: '', apiKey: '' }, targetFolders: [] })
  );
  const [saveStatus, setSaveStatus] = useState<SaveStatus>('idle');
  const [watcherStatus, setWatcherStatus] = useState<WatcherStatus>('stopped');
  const [logs, setLogs] = useState<string[]>([]);
  const [startupRegistered, setStartupRegistered] = useState<boolean>(false);
  const [startupLoading, setStartupLoading] = useState<boolean>(false);

  useEffect(() => {
    LoadConfig().then(setCfg);
    IsStartupRegistered().then(setStartupRegistered);
  }, []);

  const handleSave = async () => {
    try {
      await SaveConfig(cfg);
      setSaveStatus('saved');
      setTimeout(() => setSaveStatus('idle'), 2000);
    } catch {
      setSaveStatus('error');
      setTimeout(() => setSaveStatus('idle'), 2000);
    }
  };

  const handleAddFolder = async () => {
    const folder = await SelectFolder();
    if (!folder) return;
    if (cfg.targetFolders?.includes(folder)) return;
    const updated = config.Config.createFrom({
      ...cfg,
      targetFolders: [...(cfg.targetFolders ?? []), folder],
    });
    setCfg(updated);
    await SaveConfig(updated);
  };

  const handleRemoveFolder = async (folder: string) => {
    const updated = config.Config.createFrom({
      ...cfg,
      targetFolders: cfg.targetFolders?.filter((f) => f !== folder) ?? [],
    });
    setCfg(updated);
    await SaveConfig(updated);
  };

  const handleStartWatcher = () => {
    StartWatcher();
    setWatcherStatus('running');
    setLogs((prev) => [...prev, `[${now()}] Watcher started.`]);
  };

  const handleStopWatcher = () => {
    StopWatcher();
    setWatcherStatus('stopped');
    setLogs((prev) => [...prev, `[${now()}] Watcher stopped.`]);
  };

  const handleSyncNow = () => {
    SyncNow();
    setLogs((prev) => [...prev, `[${now()}] Manual sync triggered.`]);
  };

  const handleToggleStartup = async () => {
    setStartupLoading(true);
    try {
      if (startupRegistered) {
        await UnRegisterStartup();
        setStartupRegistered(false);
      } else {
        await RegisterStartup();
        setStartupRegistered(true);
      }
    } finally {
      setStartupLoading(false);
    }
  };

  const navItems: { id: Page; label: string; icon: string }[] = [
    { id: 'connection', label: 'Connection Settings', icon: 'settings_input_component' },
    { id: 'folders', label: 'Folder Management', icon: 'folder_shared' },
    { id: 'status', label: 'Sync Status', icon: 'sync_saved_locally' },
    { id: 'startup', label: 'Startup Settings', icon: 'play_circle' },
  ];

  return (
    <div className="flex h-screen w-screen bg-surface text-on-surface font-body overflow-hidden">
      {/* Sidebar */}
      <nav className="w-64 h-full bg-surface-container-low flex flex-col py-8 px-4 shrink-0">
        <div className="mb-10 px-2">
          <h1 className="text-base font-bold text-primary uppercase tracking-wide">Immich Windows Sync</h1>
          <p className="text-[10px] text-on-surface-variant uppercase tracking-widest mt-1 opacity-60">v{__APP_VERSION__}</p>
        </div>

        <div className="flex-1 space-y-1">
          {navItems.map((item) => (
            <button
              key={item.id}
              onClick={() => setActivePage(item.id)}
              className={`w-full flex items-center gap-3 px-4 py-3 rounded-lg text-sm font-medium transition-all duration-200 ${
                activePage === item.id
                  ? 'bg-surface-container text-primary border-l-2 border-primary font-semibold'
                  : 'text-on-surface-variant hover:text-primary hover:bg-surface-container'
              }`}
            >
              <span className="material-symbols-outlined text-[20px]">{item.icon}</span>
              <span>{item.label}</span>
            </button>
          ))}
        </div>
      </nav>

      {/* Main Content */}
      <main className="flex-1 flex flex-col overflow-hidden">
        {/* Header */}
        <header className="h-16 flex items-center px-10 bg-surface border-b border-outline-variant/10 shrink-0">
          <h2 className="text-xl font-bold tracking-tight text-primary">
            {navItems.find((i) => i.id === activePage)?.label}
          </h2>
        </header>

        {/* Page Content */}
        <div className="flex-1 overflow-y-auto p-10">
          {activePage === 'connection' && (
            <ConnectionPage cfg={cfg} setCfg={setCfg} onSave={handleSave} saveStatus={saveStatus} />
          )}
          {activePage === 'folders' && (
            <FolderManagementPage
              folders={cfg.targetFolders ?? []}
              onAdd={handleAddFolder}
              onRemove={handleRemoveFolder}
            />
          )}
          {activePage === 'status' && (
            <SyncStatusPage
              watcherStatus={watcherStatus}
              logs={logs}
              onStart={handleStartWatcher}
              onStop={handleStopWatcher}
              onSyncNow={handleSyncNow}
            />
          )}
          {activePage === 'startup' && (
            <StartupPage
              registered={startupRegistered}
              loading={startupLoading}
              onToggle={handleToggleStartup}
            />
          )}
        </div>
      </main>
    </div>
  );
}

function now() {
  return new Date().toLocaleTimeString();
}

function ConnectionPage({
  cfg,
  setCfg,
  onSave,
  saveStatus,
}: {
  cfg: config.Config;
  setCfg: (cfg: config.Config) => void;
  onSave: () => void;
  saveStatus: SaveStatus;
}) {
  const [showApiKey, setShowApiKey] = useState(false);

  const updateImmich = (field: keyof config.ImmichConfig, value: string) => {
    setCfg(
      config.Config.createFrom({
        ...cfg,
        immich: { ...cfg.immich, [field]: value },
      })
    );
  };

  return (
    <div className="w-full">
      <p className="text-on-surface-variant text-sm mb-8">
        Configure your Immich server connection.
      </p>

      <div className="bg-surface-container rounded-xl p-8 space-y-6">
        <div className="space-y-2">
          <label className="text-[10px] uppercase tracking-widest font-bold text-primary opacity-80">
            Server URL
          </label>
          <div className="relative">
            <input
              type="text"
              value={cfg.immich?.serverURL ?? ''}
              onChange={(e) => updateImmich('serverURL', e.target.value)}
              placeholder="http://192.168.1.x:2283"
              className="w-full bg-surface-container-lowest border border-outline-variant/20 rounded-lg px-4 py-3 text-on-surface text-sm font-mono focus:outline-none focus:border-primary focus:ring-1 focus:ring-primary/20 transition-all"
            />
            <span className="absolute right-4 top-1/2 -translate-y-1/2 material-symbols-outlined text-on-surface-variant/40 text-[18px]">
              link
            </span>
          </div>
        </div>

        <div className="space-y-2">
          <label className="text-[10px] uppercase tracking-widest font-bold text-primary opacity-80">
            API Key
          </label>
          <div className="relative">
            <input
              type={showApiKey ? 'text' : 'password'}
              value={cfg.immich?.apiKey ?? ''}
              onChange={(e) => updateImmich('apiKey', e.target.value)}
              placeholder="Your Immich API key"
              className="w-full bg-surface-container-lowest border border-outline-variant/20 rounded-lg px-4 py-3 text-on-surface text-sm font-mono focus:outline-none focus:border-primary focus:ring-1 focus:ring-primary/20 transition-all"
            />
            <button
              type="button"
              onClick={() => setShowApiKey((v) => !v)}
              className="absolute right-4 top-1/2 -translate-y-1/2 material-symbols-outlined text-on-surface-variant hover:text-primary transition-colors text-[18px]"
            >
              {showApiKey ? 'visibility_off' : 'visibility'}
            </button>
          </div>
        </div>

        <div className="flex items-center justify-end gap-4 pt-2">
          {saveStatus === 'saved' && (
            <span className="text-xs text-secondary flex items-center gap-1">
              <span className="material-symbols-outlined text-[16px]">check_circle</span>
              Saved
            </span>
          )}
          {saveStatus === 'error' && (
            <span className="text-xs text-error flex items-center gap-1">
              <span className="material-symbols-outlined text-[16px]">error</span>
              Failed to save
            </span>
          )}
          <button
            onClick={onSave}
            className="bg-gradient-to-br from-primary to-primary-container text-on-primary-fixed px-8 py-2.5 rounded-lg font-bold text-sm shadow-lg shadow-primary/10 hover:shadow-primary/20 active:scale-95 transition-all"
          >
            Save Changes
          </button>
        </div>
      </div>
    </div>
  );
}

function FolderManagementPage({
  folders,
  onAdd,
  onRemove,
}: {
  folders: string[];
  onAdd: () => void;
  onRemove: (folder: string) => void;
}) {
  return (
    <div className="w-full">
      <p className="text-on-surface-variant text-sm mb-8">
        Select folders to monitor and sync to Immich.
      </p>

      <div className="bg-surface-container rounded-xl p-8 space-y-6">
        <div className="space-y-2">
          {folders.length === 0 ? (
            <p className="text-on-surface-variant text-sm text-center py-6">
              No folders added yet.
            </p>
          ) : (
            folders.map((folder) => (
              <div
                key={folder}
                className="flex items-center justify-between bg-surface-container-lowest border border-outline-variant/20 rounded-lg px-4 py-3"
              >
                <div className="flex items-center gap-3">
                  <span className="material-symbols-outlined text-primary text-[20px] shrink-0">folder</span>
                  <span className="text-sm font-mono text-on-surface break-all">{folder}</span>
                </div>
                <button
                  onClick={() => onRemove(folder)}
                  className="material-symbols-outlined text-on-surface-variant hover:text-error transition-colors text-[20px] shrink-0 ml-4"
                >
                  delete
                </button>
              </div>
            ))
          )}
        </div>

        <button
          onClick={onAdd}
          className="w-full flex items-center justify-center gap-2 border border-dashed border-outline-variant/40 hover:border-primary text-on-surface-variant hover:text-primary rounded-lg py-3 text-sm font-medium transition-all"
        >
          <span className="material-symbols-outlined text-[20px]">add</span>
          Add Folder
        </button>
      </div>
    </div>
  );
}

function SyncStatusPage({
  watcherStatus,
  logs,
  onStart,
  onStop,
  onSyncNow,
}: {
  watcherStatus: WatcherStatus;
  logs: string[];
  onStart: () => void;
  onStop: () => void;
  onSyncNow: () => void;
}) {
  return (
    <div className="w-full space-y-6">
      {/* Watcher Controls */}
      <div className="bg-surface-container rounded-xl p-6 space-y-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className={`h-2 w-2 rounded-full ${watcherStatus === 'running' ? 'bg-secondary shadow-[0_0_8px_rgba(23,192,253,0.6)]' : 'bg-outline'}`} />
            <span className="text-sm font-medium text-on-surface">
              {watcherStatus === 'running' ? 'Watcher Running' : 'Watcher Stopped'}
            </span>
          </div>
          <div className="flex items-center gap-3">
            <button
              onClick={onSyncNow}
              disabled={watcherStatus !== 'running'}
              className="flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium bg-surface-container-highest text-on-surface hover:bg-surface-bright disabled:opacity-30 disabled:cursor-not-allowed transition-all"
            >
              <span className="material-symbols-outlined text-[18px]">sync</span>
              Sync Now
            </button>
            {watcherStatus === 'stopped' ? (
              <button
                onClick={onStart}
                className="flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-bold bg-gradient-to-br from-primary to-primary-container text-on-primary-fixed active:scale-95 transition-all"
              >
                <span className="material-symbols-outlined text-[18px]">play_arrow</span>
                Start
              </button>
            ) : (
              <button
                onClick={onStop}
                className="flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-bold bg-surface-container-highest text-error hover:bg-error-container/20 active:scale-95 transition-all"
              >
                <span className="material-symbols-outlined text-[18px]">stop</span>
                Stop
              </button>
            )}
          </div>
        </div>
      </div>

      {/* Log Area */}
      <div className="bg-surface-container rounded-xl p-6 space-y-3">
        <span className="text-[10px] uppercase tracking-widest font-bold text-on-surface-variant">
          Logs
        </span>
        <div className="bg-surface-container-lowest rounded-lg p-4 h-64 overflow-y-auto font-mono text-xs text-on-surface-variant space-y-1">
          {logs.length === 0 ? (
            <p className="text-center pt-8">No logs yet.</p>
          ) : (
            logs.map((log, i) => (
              <p key={i}>{log}</p>
            ))
          )}
        </div>
      </div>
    </div>
  );
}

function StartupPage({
  registered,
  loading,
  onToggle,
}: {
  registered: boolean;
  loading: boolean;
  onToggle: () => void;
}) {
  return (
    <div className="w-full">
      <p className="text-on-surface-variant text-sm mb-8">
        Configure whether the application starts automatically when Windows boots.
      </p>

      <div className="bg-surface-container rounded-xl p-8 space-y-6">
        <div className="flex items-center justify-between">
          <div className="space-y-1">
            <p className="text-sm font-semibold text-on-surface">Launch at Windows startup</p>
            <p className="text-xs text-on-surface-variant">
              Automatically start immich-sync when you log in to Windows.
            </p>
          </div>
          <button
            onClick={onToggle}
            disabled={loading}
            aria-pressed={registered}
            className={`relative inline-flex h-7 w-12 shrink-0 items-center rounded-full border-2 transition-colors duration-200 focus:outline-none disabled:opacity-50 disabled:cursor-not-allowed ${
              registered
                ? 'bg-primary border-primary'
                : 'bg-surface-container-highest border-outline-variant'
            }`}
          >
            <span
              className={`inline-block h-4 w-4 transform rounded-full bg-white shadow transition-transform duration-200 ${
                registered ? 'translate-x-5' : 'translate-x-1'
              }`}
            />
          </button>
        </div>

        <div className={`flex items-center gap-3 rounded-lg px-4 py-3 text-sm ${
          registered
            ? 'bg-secondary/10 text-secondary'
            : 'bg-surface-container-highest text-on-surface-variant'
        }`}>
          <span className="material-symbols-outlined text-[18px] shrink-0">
            {registered ? 'check_circle' : 'radio_button_unchecked'}
          </span>
          <span>
            {registered
              ? 'Registered — immich-sync will start automatically on login.'
              : 'Not registered — immich-sync will not start automatically.'}
          </span>
        </div>
      </div>
    </div>
  );
}

export default App;
