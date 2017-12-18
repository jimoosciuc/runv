package libvirt

import (
	"testing"
	"time"
)

func testStorageVolXML(volName, poolPath string) string {
	defName := volName
	if defName == "" {
		defName = time.Now().String()
	}
	return `<volume>
        <name>` + defName + `</name>
        <allocation>0</allocation>
        <capacity unit="M">10</capacity>
        <target>
          <path>` + "/" + poolPath + "/" + defName + `</path>
          <permissions>
            <owner>107</owner>
            <group>107</group>
            <mode>0744</mode>
            <label>testLabel0</label>
          </permissions>
        </target>
      </volume>`
}

func TestStorageVolGetInfo(t *testing.T) {
	pool, conn := buildTestStoragePool("")
	defer func() {
		pool.Undefine()
		pool.Free()
		if res, _ := conn.CloseConnection(); res != 0 {
			t.Errorf("CloseConnection() == %d, expected 0", res)
		}
	}()
	if err := pool.Create(0); err != nil {
		t.Error(err)
		return
	}
	defer pool.Destroy()
	vol, err := pool.StorageVolCreateXML(testStorageVolXML("", "default-pool"), 0)
	if err != nil {
		t.Error(err)
		return
	}
	defer func() {
		vol.Delete(VIR_STORAGE_VOL_DELETE_NORMAL)
		vol.Free()
	}()
	if _, err := vol.GetInfo(); err != nil {
		t.Fatal(err)
	}
}

func TestStorageVolGetKey(t *testing.T) {
	pool, conn := buildTestStoragePool("")
	defer func() {
		pool.Undefine()
		pool.Free()
		if res, _ := conn.CloseConnection(); res != 0 {
			t.Errorf("CloseConnection() == %d, expected 0", res)
		}
	}()
	if err := pool.Create(0); err != nil {
		t.Error(err)
		return
	}
	defer pool.Destroy()
	vol, err := pool.StorageVolCreateXML(testStorageVolXML("", "default-pool"), 0)
	if err != nil {
		t.Error(err)
		return
	}
	defer func() {
		vol.Delete(VIR_STORAGE_VOL_DELETE_NORMAL)
		vol.Free()
	}()
	if _, err := vol.GetKey(); err != nil {
		t.Fatal(err)
	}
}

func TestStorageVolGetName(t *testing.T) {
	pool, conn := buildTestStoragePool("")
	defer func() {
		pool.Undefine()
		pool.Free()
		if res, _ := conn.CloseConnection(); res != 0 {
			t.Errorf("CloseConnection() == %d, expected 0", res)
		}
	}()
	if err := pool.Create(0); err != nil {
		t.Error(err)
		return
	}
	defer pool.Destroy()
	vol, err := pool.StorageVolCreateXML(testStorageVolXML("", "default-pool"), 0)
	if err != nil {
		t.Error(err)
		return
	}
	defer func() {
		vol.Delete(VIR_STORAGE_VOL_DELETE_NORMAL)
		vol.Free()
	}()
	if _, err := vol.GetName(); err != nil {
		t.Fatal(err)
	}
}

func TestStorageVolGetPath(t *testing.T) {
	pool, conn := buildTestStoragePool("")
	defer func() {
		pool.Undefine()
		pool.Free()
		if res, _ := conn.CloseConnection(); res != 0 {
			t.Errorf("CloseConnection() == %d, expected 0", res)
		}
	}()
	if err := pool.Create(0); err != nil {
		t.Error(err)
		return
	}
	defer pool.Destroy()
	vol, err := pool.StorageVolCreateXML(testStorageVolXML("", "default-pool"), 0)
	if err != nil {
		t.Error(err)
		return
	}
	defer func() {
		vol.Delete(VIR_STORAGE_VOL_DELETE_NORMAL)
		vol.Free()
	}()
	if _, err := vol.GetPath(); err != nil {
		t.Fatal(err)
	}
}

func TestStorageVolGetXMLDesc(t *testing.T) {
	pool, conn := buildTestStoragePool("")
	defer func() {
		pool.Undefine()
		pool.Free()
		if res, _ := conn.CloseConnection(); res != 0 {
			t.Errorf("CloseConnection() == %d, expected 0", res)
		}
	}()
	if err := pool.Create(0); err != nil {
		t.Error(err)
		return
	}
	defer pool.Destroy()
	vol, err := pool.StorageVolCreateXML(testStorageVolXML("", "default-pool"), 0)
	if err != nil {
		t.Error(err)
		return
	}
	defer func() {
		vol.Delete(VIR_STORAGE_VOL_DELETE_NORMAL)
		vol.Free()
	}()
	if _, err := vol.GetXMLDesc(0); err != nil {
		t.Fatal(err)
	}
}

func TestPoolLookupByVolume(t *testing.T) {
	pool, conn := buildTestStoragePool("")
	defer func() {
		pool.Undefine()
		pool.Free()
		if res, _ := conn.CloseConnection(); res != 0 {
			t.Errorf("CloseConnection() == %d, expected 0", res)
		}
	}()
	if err := pool.Create(0); err != nil {
		t.Error(err)
		return
	}
	defer pool.Destroy()
	vol, err := pool.StorageVolCreateXML(testStorageVolXML("", "default-pool"), 0)
	if err != nil {
		t.Error(err)
		return
	}
	defer func() {
		vol.Delete(VIR_STORAGE_VOL_DELETE_NORMAL)
		vol.Free()
	}()

	retPool, err := vol.LookupPoolByVolume()
	if err != nil {
		t.Fatal(err)
	}
	defer retPool.Free()

	poolUUID, err := pool.GetUUIDString()
	if err != nil {
		t.Fatal(err)
	}

	retPoolUUID, err := retPool.GetUUIDString()
	if err != nil {
		t.Fatal(err)
	}

	if retPoolUUID != poolUUID {
		t.Fail()
	}
}
