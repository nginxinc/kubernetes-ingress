package configs

// appProtectDosResource holds the file names of APDosPolicy and APDosLogConf resources used in an Ingress resource.
type appProtectDosResource struct {
	AppProtectDosEnable       string
	AppProtectDosLogEnable    bool
	AppProtectDosMonitor      string
	AppProtectDosName         string
	AppProtectDosAccessLogDst string
	AppProtectDosPolicyFile   string
	AppProtectDosLogConfFile  string
}

func getAppProtectDosResource(dosEx *DosEx) *appProtectDosResource {
	var dosResource appProtectDosResource
	if dosEx == nil || dosEx.DosProtected == nil {
		return nil
	}

	protected := dosEx.DosProtected
	dosResource.AppProtectDosEnable = "off"
	if protected.Spec.Enable {
		dosResource.AppProtectDosEnable = "on"
	}
	dosResource.AppProtectDosName = protected.Namespace + "/" + protected.Name + "/" + protected.Spec.Name
	dosResource.AppProtectDosMonitor = protected.Spec.ApDosMonitor
	dosResource.AppProtectDosAccessLogDst = generateDosLogDest(protected.Spec.DosAccessLogDest)

	if dosEx.DosPolicy != nil {
		dosResource.AppProtectDosPolicyFile = appProtectDosPolicyFileName(dosEx.DosPolicy.GetNamespace(), dosEx.DosPolicy.GetName())
	}

	if dosEx.DosLogConf != nil {
		log := dosEx.DosLogConf
		logConfFileName := appProtectDosLogConfFileName(log.GetNamespace(), log.GetName())
		dosResource.AppProtectDosLogConfFile = logConfFileName + " " + generateDosLogDest(protected.Spec.DosSecurityLog.DosLogDest)
		dosResource.AppProtectDosLogEnable = protected.Spec.DosSecurityLog.Enable
	}

	return &dosResource
}
