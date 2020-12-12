package test_test

import (
	"flag"
	"fmt"
	"free5gc/lib/MongoDBLibrary"
	"free5gc/lib/ngap"
	"free5gc/lib/ngap/ngapSctp"
	"free5gc/lib/path_util"
	"free5gc/src/amf/amf_service"
	"free5gc/src/app"
	"free5gc/src/ausf/ausf_service"
	"free5gc/src/nrf/nrf_service"
	"free5gc/src/nssf/nssf_service"
	"free5gc/src/pcf/pcf_service"
	"free5gc/src/smf/smf_service"
	"free5gc/src/test"
	"free5gc/src/udm/udm_service"
	"free5gc/src/udr/udr_service"
	"log"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	"git.cs.nctu.edu.tw/calee/sctp"
	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli"
	"strconv"
)

// I2CAT - Parsing arguments passed to Test methods.
var NumUes = flag.Int("NumUes", 1, "Number of Ues that need to be registered") //For TestI2catCN


var NFs = []app.NetworkFunction{
	&nrf_service.NRF{},
	&amf_service.AMF{},
	&smf_service.SMF{},
	&udr_service.UDR{},
	&pcf_service.PCF{},
	&udm_service.UDM{},
	&nssf_service.NSSF{},
	&ausf_service.AUSF{},
	//&n3iwf_service.N3IWF{},
}

func init() {
	var init bool = true

	for _, arg := range os.Args {
		if arg == "noinit" {
			init = false
		}
	}

	if init {
		app.AppInitializeWillInitialize("")
		flagSet := flag.NewFlagSet("free5gc", 0)
		flagSet.String("smfcfg", "", "SMF Config Path")
		cli := cli.NewContext(nil, flagSet, nil)
		err := cli.Set("smfcfg", path_util.Gofree5gcPath("free5gc/config/test/smfcfg.test.conf"))
		if err != nil {
			log.Fatal("SMF test config error")
			return
		}

		for _, service := range NFs {
			service.Initialize(cli)
			go service.Start()
			time.Sleep(200 * time.Millisecond)
		}
	} else {
		MongoDBLibrary.SetMongoDB("free5gc", "mongodb://127.0.0.1:27017")
		fmt.Println("MongoDB Set")
	}

}

func getNgapIp(amfIP, ranIP string, amfPort, ranPort int) (amfAddr, ranAddr *sctp.SCTPAddr, err error) {
	ips := []net.IPAddr{}
	if ip, err1 := net.ResolveIPAddr("ip", amfIP); err1 != nil {
		err = fmt.Errorf("Error resolving address '%s': %v", amfIP, err1)
		return
	} else {
		ips = append(ips, *ip)
	}
	amfAddr = &sctp.SCTPAddr{
		IPAddrs: ips,
		Port:    amfPort,
	}
	ips = []net.IPAddr{}
	if ip, err1 := net.ResolveIPAddr("ip", ranIP); err1 != nil {
		err = fmt.Errorf("Error resolving address '%s': %v", ranIP, err1)
		return
	} else {
		ips = append(ips, *ip)
	}
	ranAddr = &sctp.SCTPAddr{
		IPAddrs: ips,
		Port:    ranPort,
	}
	return
}

func conntectToAmf(amfIP, ranIP string, amfPort, ranPort int) (*sctp.SCTPConn, error) {
	amfAddr, ranAddr, err := getNgapIp(amfIP, ranIP, amfPort, ranPort)
	if err != nil {
		return nil, err
	}
	conn, err := sctp.DialSCTP("sctp", ranAddr, amfAddr)
	if err != nil {
		return nil, err
	}
	info, _ := conn.GetDefaultSentParam()
	info.PPID = ngapSctp.NGAP_PPID
	err = conn.SetDefaultSentParam(info)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func TestNGSetup(t *testing.T) {
	var n int
	var sendMsg []byte
	var recvMsg = make([]byte, 2048)

	// RAN connect to AMF
	conn, err := conntectToAmf("127.0.0.1", "127.0.0.1", 38412, 9487)
	assert.Nil(t, err)

	// send NGSetupRequest Msg
	sendMsg, err = test.GetNGSetupRequest([]byte("\x00\x01\x02"), 24, "free5gc")
	assert.Nil(t, err)
	_, err = conn.Write(sendMsg)
	assert.Nil(t, err)

	// receive NGSetupResponse Msg
	n, err = conn.Read(recvMsg)
	assert.Nil(t, err)
	_, err = ngap.Decoder(recvMsg[:n])
	assert.Nil(t, err)

	// close Connection
	conn.Close()
}

func TestCN(t *testing.T) {
	// New UE
	ue := test.NewRanUeContext("imsi-2089300007487", 1, test.ALG_CIPHERING_128_NEA2, test.ALG_INTEGRITY_128_NIA2)
	// ue := test.NewRanUeContext("imsi-2089300007487", 1, test.ALG_CIPHERING_128_NEA0, test.ALG_INTEGRITY_128_NIA0)
	ue.AmfUeNgapId = 1
	ue.AuthenticationSubs = getAuthSubscription()
	// insert UE data to MongoDB

	servingPlmnId := "20893"
	test.InsertAuthSubscriptionToMongoDB(ue.Supi, ue.AuthenticationSubs)
	getData := test.GetAuthSubscriptionFromMongoDB(ue.Supi)

	assert.NotNil(t, getData)
	{
		amData := getAccessAndMobilitySubscriptionData()
		test.InsertAccessAndMobilitySubscriptionDataToMongoDB(ue.Supi, amData, servingPlmnId)
		getData := test.GetAccessAndMobilitySubscriptionDataFromMongoDB(ue.Supi, servingPlmnId)
		assert.NotNil(t, getData)
	}
	{
		smfSelData := getSmfSelectionSubscriptionData()
		test.InsertSmfSelectionSubscriptionDataToMongoDB(ue.Supi, smfSelData, servingPlmnId)
		getData := test.GetSmfSelectionSubscriptionDataFromMongoDB(ue.Supi, servingPlmnId)
		assert.NotNil(t, getData)
	}
	{
		smSelData := getSessionManagementSubscriptionData()
		test.InsertSessionManagementSubscriptionDataToMongoDB(ue.Supi, servingPlmnId, smSelData)
		getData := test.GetSessionManagementDataFromMongoDB(ue.Supi, servingPlmnId)
		assert.NotNil(t, getData)
	}
	{
		amPolicyData := getAmPolicyData()
		test.InsertAmPolicyDataToMongoDB(ue.Supi, amPolicyData)
		getData := test.GetAmPolicyDataFromMongoDB(ue.Supi)
		assert.NotNil(t, getData)
	}
	{
		smPolicyData := getSmPolicyData()
		test.InsertSmPolicyDataToMongoDB(ue.Supi, smPolicyData)
		getData := test.GetSmPolicyDataFromMongoDB(ue.Supi)
		assert.NotNil(t, getData)
	}

	defer beforeClose(ue)

	wg := sync.WaitGroup{}
	wg.Add(1)
	wg.Wait()
}

func TestI2catCN(t *testing.T) {

	fmt.Println("#### TestI2catCN with NumUes = ", *NumUes)
        for i := 1;  i<=*NumUes; i++ {
		// New UE
		var UeImsi string
		UeImsi = "imsi-208930000748" + strconv.Itoa(i)
		fmt.Println("#### Creating UE context for UeImsi=", UeImsi)
		ue := test.NewRanUeContext(UeImsi, 1, test.ALG_CIPHERING_128_NEA2, test.ALG_INTEGRITY_128_NIA2)
		// ue := test.NewRanUeContext("imsi-2089300007487", 1, test.ALG_CIPHERING_128_NEA0, test.ALG_INTEGRITY_128_NIA0)
	//	ue.AmfUeNgapId = 1
		ue.AmfUeNgapId = int64(i)
		ue.AuthenticationSubs = getAuthSubscription()
		// insert UE data to MongoDB

		servingPlmnId := "20893"
		test.InsertAuthSubscriptionToMongoDB(ue.Supi, ue.AuthenticationSubs)
		getData := test.GetAuthSubscriptionFromMongoDB(ue.Supi)

		assert.NotNil(t, getData)
		{
			amData := getAccessAndMobilitySubscriptionData()
			test.InsertAccessAndMobilitySubscriptionDataToMongoDB(ue.Supi, amData, servingPlmnId)
			getData := test.GetAccessAndMobilitySubscriptionDataFromMongoDB(ue.Supi, servingPlmnId)
			assert.NotNil(t, getData)
		}
		{
			smfSelData := getSmfSelectionSubscriptionData()
			test.InsertSmfSelectionSubscriptionDataToMongoDB(ue.Supi, smfSelData, servingPlmnId)
			getData := test.GetSmfSelectionSubscriptionDataFromMongoDB(ue.Supi, servingPlmnId)
			assert.NotNil(t, getData)
		}
		{
			smSelData := getSessionManagementSubscriptionData()
			test.InsertSessionManagementSubscriptionDataToMongoDB(ue.Supi, servingPlmnId, smSelData)
			getData := test.GetSessionManagementDataFromMongoDB(ue.Supi, servingPlmnId)
			assert.NotNil(t, getData)
		}
		{
			amPolicyData := getAmPolicyData()
			test.InsertAmPolicyDataToMongoDB(ue.Supi, amPolicyData)
			getData := test.GetAmPolicyDataFromMongoDB(ue.Supi)
			assert.NotNil(t, getData)
		}
		{
			smPolicyData := getSmPolicyData()
			test.InsertSmPolicyDataToMongoDB(ue.Supi, smPolicyData)
			getData := test.GetSmPolicyDataFromMongoDB(ue.Supi)
			assert.NotNil(t, getData)
		}

		defer beforeClose(ue)

	} // closing for loop

	wg := sync.WaitGroup{}
	wg.Add(1)
	wg.Wait()
}



func beforeClose(ue *test.RanUeContext) {
	// delete test data
	test.DelAuthSubscriptionToMongoDB(ue.Supi)
	test.DelAccessAndMobilitySubscriptionDataFromMongoDB(ue.Supi, "20893")
	test.DelSmfSelectionSubscriptionDataFromMongoDB(ue.Supi, "20893")
}
