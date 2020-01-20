package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCSIVolumes_CRUD(t *testing.T) {
	t.Parallel()
	c, s, root := makeACLClient(t, nil, nil)
	defer s.Stop()
	v := c.CSIVolumes()

	// Successful empty result
	vols, qm, err := v.List(nil)
	assert.NoError(t, err)
	assert.NotEqual(t, 0, qm.LastIndex)
	assert.Equal(t, 0, len(vols))

	// Authorized QueryOpts. Use the root token to just bypass ACL details
	opts := &QueryOptions{
		Region:    "global",
		Namespace: "default",
		AuthToken: root.SecretID,
	}

	wpts := &WriteOptions{
		Region:    "global",
		Namespace: "default",
		AuthToken: root.SecretID,
	}

	// Register a volume
	id := "DEADBEEF-31B5-8F78-7986-DD404FDA0CD1"
	err = v.Register(&CSIVolume{
		ID:             id,
		Namespace:      "default",
		PluginID:       "adam",
		AccessMode:     CSIVolumeAccessModeMultiNodeSingleWriter,
		AttachmentMode: CSIVolumeAttachmentModeFilesystem,
		Topologies:     []*CSITopology{{Segments: map[string]string{"foo": "bar"}}},
	}, wpts)
	assert.NoError(t, err)

	// Successful result with volumes
	vols, qm, err = v.List(opts)
	assert.NoError(t, err)
	assert.NotEqual(t, 0, qm.LastIndex)
	assert.Equal(t, 1, len(vols))

	// Successful info query
	vol, qm, err := v.Info(id, opts)
	assert.NoError(t, err)
	assert.Equal(t, "bar", vol.Topologies[0].Segments["foo"])

	// Deregister the volume
	err = v.Deregister(id, wpts)
	assert.NoError(t, err)

	// Successful empty result
	vols, qm, err = v.List(nil)
	assert.NoError(t, err)
	assert.NotEqual(t, 0, qm.LastIndex)
	assert.Equal(t, 0, len(vols))

	// Failed info query
	vol, qm, err = v.Info(id, opts)
	assert.Error(t, err, "missing")
}

func TestCSIPlugins_viaJob(t *testing.T) {
	t.Parallel()
	c, s, root := makeACLClient(t, nil, nil)
	defer s.Stop()
	p := c.CSIPlugins()

	// Successful empty result
	plugs, qm, err := p.List(nil)
	assert.NoError(t, err)
	assert.NotEqual(t, 0, qm.LastIndex)
	assert.Equal(t, 0, len(plugs))

	// Authorized QueryOpts. Use the root token to just bypass ACL details
	opts := &QueryOptions{
		Region:    "global",
		Namespace: "default",
		AuthToken: root.SecretID,
	}

	wpts := &WriteOptions{
		Region:    "global",
		Namespace: "default",
		AuthToken: root.SecretID,
	}

	// Register a plugin job
	j := c.Jobs()
	job := testJob()
	job.Namespace = stringToPtr("default")
	job.TaskGroups[0].Tasks[0].CSIPluginConfig = &TaskCSIPluginConfig{
		ID:       "foo",
		Type:     "monolith",
		MountDir: "/not-empty",
	}
	_, _, err = j.Register(job, wpts)
	require.NoError(t, err)

	// Successful result with the plugin
	plugs, qm, err = p.List(opts)
	assert.NoError(t, err)
	assert.NotEqual(t, 0, qm.LastIndex)
	assert.Equal(t, 1, len(plugs))

	// Successful info query
	plug, qm, err := p.Info("foo", opts)
	require.NoError(t, err)
	assert.Equal(t, *job.ID, *plug.Jobs[*job.Namespace][*job.ID].ID)
}
