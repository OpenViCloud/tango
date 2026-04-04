package main

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
	"github.com/spf13/cobra"
)

var swarmCmd = &cobra.Command{
	Use:   "swarm",
	Short: "Manage Docker Swarm cluster",
}

// ── tango swarm init ──────────────────────────────────────────────────────────

var swarmInitAdvertiseAddr string

var swarmInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize this node as a swarm manager",
	RunE: func(cmd *cobra.Command, args []string) error {
		cli, err := newDockerClient()
		if err != nil {
			return err
		}
		defer cli.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		resp, err := cli.SwarmInit(ctx, swarm.InitRequest{
			ListenAddr:    "0.0.0.0:2377",
			AdvertiseAddr: swarmInitAdvertiseAddr,
		})
		if err != nil {
			return fmt.Errorf("swarm init: %w", err)
		}

		fmt.Println("Swarm initialized.")
		fmt.Println("Node ID:", resp)
		fmt.Println()
		fmt.Println("To add workers to this swarm, run:")
		fmt.Printf("  tango swarm token worker\n")
		return nil
	},
}

// ── tango swarm token ─────────────────────────────────────────────────────────

var swarmTokenCmd = &cobra.Command{
	Use:   "token [worker|manager]",
	Short: "Show the join token for workers or managers",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		role := args[0]
		if role != "worker" && role != "manager" {
			return fmt.Errorf("role must be 'worker' or 'manager'")
		}

		cli, err := newDockerClient()
		if err != nil {
			return err
		}
		defer cli.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		sw, err := cli.SwarmInspect(ctx)
		if err != nil {
			return fmt.Errorf("inspect swarm: %w", err)
		}

		info, err := cli.Info(ctx)
		if err != nil {
			return fmt.Errorf("docker info: %w", err)
		}

		token := sw.JoinTokens.Worker
		if role == "manager" {
			token = sw.JoinTokens.Manager
		}

		managerAddr := ""
		if info.Swarm.NodeAddr != "" {
			managerAddr = fmt.Sprintf("%s:2377", info.Swarm.NodeAddr)
		}

		fmt.Printf("Join token (%s):\n\n", role)
		fmt.Printf("  tango swarm join --token %s %s\n", token, managerAddr)
		return nil
	},
}

// ── tango swarm join ──────────────────────────────────────────────────────────

var swarmJoinToken string

var swarmJoinCmd = &cobra.Command{
	Use:   "join <manager-addr>",
	Short: "Join this node to an existing swarm as a worker",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		managerAddr := args[0]

		if swarmJoinToken == "" {
			return fmt.Errorf("--token is required")
		}

		cli, err := newDockerClient()
		if err != nil {
			return err
		}
		defer cli.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := cli.SwarmJoin(ctx, swarm.JoinRequest{
			ListenAddr:  "0.0.0.0:2377",
			JoinToken:   swarmJoinToken,
			RemoteAddrs: []string{managerAddr},
		}); err != nil {
			return fmt.Errorf("swarm join: %w", err)
		}

		fmt.Println("Node joined the swarm as a worker.")
		return nil
	},
}

// ── tango swarm leave ─────────────────────────────────────────────────────────

var swarmLeaveForce bool

var swarmLeaveCmd = &cobra.Command{
	Use:   "leave",
	Short: "Remove this node from the swarm",
	RunE: func(cmd *cobra.Command, args []string) error {
		cli, err := newDockerClient()
		if err != nil {
			return err
		}
		defer cli.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		if err := cli.SwarmLeave(ctx, swarmLeaveForce); err != nil {
			return fmt.Errorf("swarm leave: %w", err)
		}

		fmt.Println("Node left the swarm.")
		return nil
	},
}

// ── tango swarm nodes ─────────────────────────────────────────────────────────

var swarmNodesCmd = &cobra.Command{
	Use:   "nodes",
	Short: "List all nodes in the swarm",
	RunE: func(cmd *cobra.Command, args []string) error {
		cli, err := newDockerClient()
		if err != nil {
			return err
		}
		defer cli.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		nodes, err := cli.NodeList(ctx, swarm.NodeListOptions{})
		if err != nil {
			return fmt.Errorf("list nodes: %w", err)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tHOSTNAME\tROLE\tSTATUS\tAVAILABILITY")
		for _, n := range nodes {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				n.ID[:12],
				n.Description.Hostname,
				string(n.Spec.Role),
				string(n.Status.State),
				string(n.Spec.Availability),
			)
		}
		w.Flush()
		return nil
	},
}

// ── helpers ───────────────────────────────────────────────────────────────────

func newDockerClient() (*client.Client, error) {
	cli, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, fmt.Errorf("connect to Docker: %w", err)
	}
	return cli, nil
}

func init() {
	swarmInitCmd.Flags().StringVar(&swarmInitAdvertiseAddr, "advertise-addr", "", "Externally reachable address for this manager (e.g. 1.2.3.4)")
	swarmJoinCmd.Flags().StringVar(&swarmJoinToken, "token", "", "Join token obtained from 'tango swarm token worker'")
	swarmLeaveCmd.Flags().BoolVar(&swarmLeaveForce, "force", false, "Force leave even if this is the last manager")

	swarmCmd.AddCommand(swarmInitCmd, swarmTokenCmd, swarmJoinCmd, swarmLeaveCmd, swarmNodesCmd)
}
