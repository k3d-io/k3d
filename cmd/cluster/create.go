// ...

func (c *ClusterCreateCmd) run() error {
    // ...

    // Start the loadbalancer container
    if err := c.startLoadbalancer(); err != nil {
        return err
    }

    // Wait for the loadbalancer to become ready
    if err := c.waitForLoadbalancer(); err != nil {
        return err
    }

    // ...

    return nil
}

func (c *ClusterCreateCmd) startLoadbalancer() error {
    // ...

    // Start the confd container
    if err := c.startConfd(); err != nil {
        return err
    }

    // Wait for the confd container to generate the nginx config
    if err := c.waitForConfd(); err != nil {
        return err
    }

    // Start the nginx container
    if err := c.startNginx(); err != nil {
        return err
    }

    // Wait for the nginx container to write a valid PID to /var/run/nginx.pid
    if err := c.waitForNginxPid(); err != nil {
        return err
    }

    // Run nginx -s reload
    if err := c.reloadNginx(); err != nil {
        return err
    }

    return nil
}

func (c *ClusterCreateCmd) waitForNginxPid() error {
    // Wait for the nginx container to write a valid PID to /var/run/nginx.pid
    for {
        pid, err := c.getNginxPid()
        if err != nil {
            return err
        }
        if pid != "" {
            break
        }
        time.Sleep(100 * time.Millisecond)
    }
    return nil
}

func (c *ClusterCreateCmd) getNginxPid() (string, error) {
    // Get the PID from the /var/run/nginx.pid file
    pidFile := "/var/run/nginx.pid"
    contents, err := ioutil.ReadFile(pidFile)
    if err != nil {
        return "", err
    }
    pid := strings.TrimSpace(string(contents))
    return pid, nil
}

// ...